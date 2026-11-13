package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/incentive"
	"github.com/haydary1986/turath-research/backend/internal/service/performance"
	"github.com/haydary1986/turath-research/backend/internal/service/promotion"
)

type ScientificAffairsHandler struct {
	db            *pgxpool.Pool
	eligibility   *promotion.EligibilityService
	incentiveCalc *incentive.Calculator
	perfEval      *performance.Evaluator
}

func NewScientificAffairsHandler(db *pgxpool.Pool) *ScientificAffairsHandler {
	return &ScientificAffairsHandler{
		db:            db,
		eligibility:   promotion.NewEligibilityService(db),
		incentiveCalc: incentive.NewCalculator(db),
		perfEval:      performance.NewEvaluator(db),
	}
}

// ══════════════════════════════════════════════════
// Promotions
// ══════════════════════════════════════════════════

// CheckEligibility returns promotion eligibility for a researcher.
func (h *ScientificAffairsHandler) CheckEligibility(w http.ResponseWriter, r *http.Request) {
	researcherID := chi.URLParam(r, "id")
	institutionID := r.URL.Query().Get("institution_id")
	if institutionID == "" {
		institutionID = "11111111-1111-1111-1111-111111111111" // default
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if !requireActsFor(ctx, w, r, h.db, researcherID) {
		return
	}

	result, err := h.eligibility.CheckEligibility(ctx, researcherID, institutionID)
	if err != nil {
		model.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	model.RespondJSON(w, http.StatusOK, result)
}

// SubmitPromotionApplication creates a new promotion application.
func (h *ScientificAffairsHandler) SubmitPromotionApplication(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResearcherID string `json:"researcher_id"`
		CurrentRank  string `json:"current_rank"`
		TargetRank   string `json:"target_rank"`
		DocumentsURL string `json:"documents_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !requireActsFor(ctx, w, r, h.db, req.ResearcherID) {
		return
	}

	id := uuid.New().String()
	_, err := h.db.Exec(ctx, `
		INSERT INTO promotion_applications
			(id, researcher_id, current_rank, target_rank, status, documents_url, submitted_at)
		VALUES ($1, $2, $3::academic_rank, $4::academic_rank, 'submitted', $5, NOW())
	`, id, req.ResearcherID, req.CurrentRank, req.TargetRank, req.DocumentsURL)
	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "Failed to submit application: "+err.Error())
		return
	}

	model.RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// ListPromotionApplications returns all promotion applications.
func (h *ScientificAffairsHandler) ListPromotionApplications(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := r.URL.Query().Get("status")
	query := `
		SELECT pa.id, pa.researcher_id, r.name_ar, pa.current_rank, pa.target_rank,
		       pa.eligibility_score, pa.status, pa.submitted_at, pa.created_at
		FROM promotion_applications pa
		JOIN researchers r ON r.id = pa.researcher_id
	`
	args := []any{}
	if status != "" {
		query += ` WHERE pa.status = $1::promotion_status`
		args = append(args, status)
	}
	query += ` ORDER BY pa.created_at DESC LIMIT 50`

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch applications")
		return
	}
	defer rows.Close()

	type appRow struct {
		ID               string  `json:"id"`
		ResearcherID     string  `json:"researcher_id"`
		ResearcherName   string  `json:"researcher_name"`
		CurrentRank      string  `json:"current_rank"`
		TargetRank       string  `json:"target_rank"`
		EligibilityScore float64 `json:"eligibility_score"`
		Status           string  `json:"status"`
		SubmittedAt      *string `json:"submitted_at"`
		CreatedAt        string  `json:"created_at"`
	}

	apps := []appRow{}
	for rows.Next() {
		var a appRow
		var submittedAt *time.Time
		var createdAt time.Time
		rows.Scan(&a.ID, &a.ResearcherID, &a.ResearcherName, &a.CurrentRank, &a.TargetRank,
			&a.EligibilityScore, &a.Status, &submittedAt, &createdAt)
		if submittedAt != nil {
			s := submittedAt.Format(time.RFC3339)
			a.SubmittedAt = &s
		}
		a.CreatedAt = createdAt.Format(time.RFC3339)
		apps = append(apps, a)
	}

	model.RespondJSON(w, http.StatusOK, apps)
}

// ══════════════════════════════════════════════════
// Incentives
// ══════════════════════════════════════════════════

// CalculateIncentives returns calculated incentives for a researcher.
func (h *ScientificAffairsHandler) CalculateIncentives(w http.ResponseWriter, r *http.Request) {
	researcherID := chi.URLParam(r, "id")
	institutionID := r.URL.Query().Get("institution_id")
	if institutionID == "" {
		institutionID = "11111111-1111-1111-1111-111111111111"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if !requireActsFor(ctx, w, r, h.db, researcherID) {
		return
	}

	results, err := h.incentiveCalc.CalculateForResearcher(ctx, researcherID, institutionID)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	model.RespondJSON(w, http.StatusOK, results)
}

// ListIncentiveRules returns the current incentive rules.
func (h *ScientificAffairsHandler) ListIncentiveRules(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT id, journal_classification, base_amount, corresponding_bonus,
		       max_authors_share, effective_from, effective_to
		FROM incentive_rules
		WHERE effective_from <= CURRENT_DATE AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)
		ORDER BY journal_classification
	`)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch rules")
		return
	}
	defer rows.Close()

	type ruleRow struct {
		ID                    string  `json:"id"`
		JournalClassification string  `json:"journal_classification"`
		BaseAmount            float64 `json:"base_amount"`
		CorrespondingBonus    float64 `json:"corresponding_bonus"`
		MaxAuthorsShare       int     `json:"max_authors_share"`
		EffectiveFrom         string  `json:"effective_from"`
		EffectiveTo           *string `json:"effective_to"`
	}

	rules := []ruleRow{}
	for rows.Next() {
		var ru ruleRow
		var from time.Time
		var to *time.Time
		rows.Scan(&ru.ID, &ru.JournalClassification, &ru.BaseAmount, &ru.CorrespondingBonus,
			&ru.MaxAuthorsShare, &from, &to)
		ru.EffectiveFrom = from.Format("2006-01-02")
		if to != nil {
			s := to.Format("2006-01-02")
			ru.EffectiveTo = &s
		}
		rules = append(rules, ru)
	}
	model.RespondJSON(w, http.StatusOK, rules)
}

// ══════════════════════════════════════════════════
// Performance Reviews
// ══════════════════════════════════════════════════

// GeneratePerformanceReviews generates reviews for a department.
func (h *ScientificAffairsHandler) GeneratePerformanceReviews(w http.ResponseWriter, r *http.Request) {
	departmentID := chi.URLParam(r, "deptId")
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	if year == 0 {
		year = time.Now().Year()
	}
	institutionID := r.URL.Query().Get("institution_id")
	if institutionID == "" {
		institutionID = "11111111-1111-1111-1111-111111111111"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	reviews, err := h.perfEval.GenerateForDepartment(ctx, departmentID, year, institutionID)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	model.RespondJSON(w, http.StatusOK, reviews)
}

// ══════════════════════════════════════════════════
// Sabbaticals
// ══════════════════════════════════════════════════

func (h *ScientificAffairsHandler) SubmitSabbatical(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResearcherID       string `json:"researcher_id"`
		Destination        string `json:"destination"`
		DestinationCountry string `json:"destination_country"`
		ResearchPlan       string `json:"research_plan"`
		StartDate          string `json:"start_date"`
		EndDate            string `json:"end_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !requireActsFor(ctx, w, r, h.db, req.ResearcherID) {
		return
	}

	id := uuid.New().String()
	_, err := h.db.Exec(ctx, `
		INSERT INTO sabbatical_requests (id, researcher_id, destination, destination_country, research_plan, start_date, end_date, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'submitted')
	`, id, req.ResearcherID, req.Destination, req.DestinationCountry, req.ResearchPlan, req.StartDate, req.EndDate)
	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "Failed to submit sabbatical request")
		return
	}
	model.RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *ScientificAffairsHandler) ListSabbaticals(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT sr.id, r.name_ar, sr.destination, sr.destination_country, sr.start_date, sr.end_date, sr.status
		FROM sabbatical_requests sr JOIN researchers r ON r.id = sr.researcher_id
		ORDER BY sr.created_at DESC LIMIT 50
	`)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch")
		return
	}
	defer rows.Close()

	type row struct {
		ID          string `json:"id"`
		Researcher  string `json:"researcher_name"`
		Destination string `json:"destination"`
		Country     string `json:"country"`
		StartDate   string `json:"start_date"`
		EndDate     string `json:"end_date"`
		Status      string `json:"status"`
	}
	items := []row{}
	for rows.Next() {
		var i row
		var sd, ed time.Time
		rows.Scan(&i.ID, &i.Researcher, &i.Destination, &i.Country, &sd, &ed, &i.Status)
		i.StartDate = sd.Format("2006-01-02")
		i.EndDate = ed.Format("2006-01-02")
		items = append(items, i)
	}
	model.RespondJSON(w, http.StatusOK, items)
}

// ══════════════════════════════════════════════════
// Conferences
// ══════════════════════════════════════════════════

func (h *ScientificAffairsHandler) SubmitConference(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResearcherID      string  `json:"researcher_id"`
		ConferenceName    string  `json:"conference_name"`
		Location          string  `json:"conference_location"`
		Country           string  `json:"conference_country"`
		StartDate         string  `json:"start_date"`
		EndDate           string  `json:"end_date"`
		PaperTitle        string  `json:"paper_title"`
		ParticipationType string  `json:"participation_type"`
		IsInternational   bool    `json:"is_international"`
		EstimatedCost     float64 `json:"estimated_cost"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.RespondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if !requireActsFor(ctx, w, r, h.db, req.ResearcherID) {
		return
	}

	pType := "presenter"
	if req.ParticipationType != "" {
		pType = req.ParticipationType
	}

	id := uuid.New().String()
	_, err := h.db.Exec(ctx, `
		INSERT INTO conference_participations
			(id, researcher_id, conference_name, conference_location, conference_country,
			 conference_date_start, conference_date_end, paper_title,
			 participation_type, is_international, estimated_cost, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::participation_type, $10, $11, 'submitted')
	`, id, req.ResearcherID, req.ConferenceName, req.Location, req.Country,
		req.StartDate, req.EndDate, req.PaperTitle, pType, req.IsInternational, req.EstimatedCost)
	if err != nil {
		model.RespondError(w, http.StatusBadRequest, "Failed to submit conference request")
		return
	}
	model.RespondJSON(w, http.StatusCreated, map[string]string{"id": id})
}

// ══════════════════════════════════════════════════
// Agreements
// ══════════════════════════════════════════════════

func (h *ScientificAffairsHandler) ListAgreements(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.Query(ctx, `
		SELECT id, partner_name_ar, COALESCE(partner_name_en, ''), agreement_type, title_ar,
		       start_date, end_date, status, outputs_count
		FROM agreements ORDER BY created_at DESC LIMIT 50
	`)
	if err != nil {
		model.RespondError(w, http.StatusInternalServerError, "Failed to fetch")
		return
	}
	defer rows.Close()

	type row struct {
		ID        string `json:"id"`
		PartnerAr string `json:"partner_name_ar"`
		PartnerEn string `json:"partner_name_en"`
		Type      string `json:"agreement_type"`
		TitleAr   string `json:"title_ar"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
		Status    string `json:"status"`
		Outputs   int    `json:"outputs_count"`
	}
	items := []row{}
	for rows.Next() {
		var i row
		var sd, ed *time.Time
		rows.Scan(&i.ID, &i.PartnerAr, &i.PartnerEn, &i.Type, &i.TitleAr, &sd, &ed, &i.Status, &i.Outputs)
		if sd != nil {
			i.StartDate = sd.Format("2006-01-02")
		}
		if ed != nil {
			i.EndDate = ed.Format("2006-01-02")
		}
		items = append(items, i)
	}
	model.RespondJSON(w, http.StatusOK, items)
}
