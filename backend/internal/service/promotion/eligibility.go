package promotion

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EligibilityService checks if a researcher meets promotion requirements.
type EligibilityService struct {
	db *pgxpool.Pool
}

func NewEligibilityService(db *pgxpool.Pool) *EligibilityService {
	return &EligibilityService{db: db}
}

// EligibilityResult holds the promotion eligibility check result.
type EligibilityResult struct {
	Eligible        bool              `json:"eligible"`
	CurrentRank     string            `json:"current_rank"`
	TargetRank      string            `json:"target_rank"`
	Score           float64           `json:"score"`
	Criteria        []CriterionCheck  `json:"criteria"`
	MissingCriteria []string          `json:"missing_criteria"`
}

type CriterionCheck struct {
	Name     string  `json:"name"`
	Required float64 `json:"required"`
	Actual   float64 `json:"actual"`
	Met      bool    `json:"met"`
}

// CheckEligibility evaluates a researcher's readiness for promotion.
func (s *EligibilityService) CheckEligibility(ctx context.Context, researcherID, institutionID string) (*EligibilityResult, error) {
	// Get researcher stats
	var hIndex, totalPubs, totalCit int
	var titleAr string
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(title_ar, ''), h_index, total_publications, total_citations
		FROM researchers WHERE id = $1
	`, researcherID).Scan(&titleAr, &hIndex, &totalPubs, &totalCit)
	if err != nil {
		return nil, fmt.Errorf("researcher not found: %w", err)
	}

	// Determine current and target rank
	currentRank := mapTitleToRank(titleAr)
	targetRank := nextRank(currentRank)
	if targetRank == "" {
		return &EligibilityResult{
			Eligible:    false,
			CurrentRank: currentRank,
			TargetRank:  "none",
			Criteria:    []CriterionCheck{},
			MissingCriteria: []string{"الرتبة الحالية هي أعلى رتبة (أستاذ)"},
		}, nil
	}

	// Count Q1 and Q2 publications
	var q1Count, q2Count int
	s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM publications p
		JOIN publication_authors pa ON pa.publication_id = p.id
		JOIN journals j ON j.id = p.journal_id
		WHERE pa.researcher_id = $1 AND j.sjr_ranking = 'Q1'
	`, researcherID).Scan(&q1Count)
	s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM publications p
		JOIN publication_authors pa ON pa.publication_id = p.id
		JOIN journals j ON j.id = p.journal_id
		WHERE pa.researcher_id = $1 AND j.sjr_ranking = 'Q2'
	`, researcherID).Scan(&q2Count)

	// Get requirements
	var minQ1, minTotal, minH, minYears int
	err = s.db.QueryRow(ctx, `
		SELECT min_q1_publications, min_total_publications, min_h_index, min_years_in_rank
		FROM promotion_requirements
		WHERE institution_id = $1 AND from_rank = $2::academic_rank AND to_rank = $3::academic_rank
		LIMIT 1
	`, institutionID, currentRank, targetRank).Scan(&minQ1, &minTotal, &minH, &minYears)
	if err != nil {
		return nil, fmt.Errorf("no promotion requirements configured for %s → %s: %w", currentRank, targetRank, err)
	}

	// Build criteria checks
	criteria := []CriterionCheck{
		{Name: "أبحاث Q1", Required: float64(minQ1), Actual: float64(q1Count), Met: q1Count >= minQ1},
		{Name: "إجمالي الأبحاث", Required: float64(minTotal), Actual: float64(totalPubs), Met: totalPubs >= minTotal},
		{Name: "h-index", Required: float64(minH), Actual: float64(hIndex), Met: hIndex >= minH},
	}

	// Calculate score (percentage of requirements met)
	metCount := 0
	missing := []string{}
	for _, c := range criteria {
		if c.Met {
			metCount++
		} else {
			missing = append(missing, fmt.Sprintf("%s: مطلوب %.0f، الفعلي %.0f", c.Name, c.Required, c.Actual))
		}
	}

	score := float64(metCount) / float64(len(criteria)) * 100

	return &EligibilityResult{
		Eligible:        metCount == len(criteria),
		CurrentRank:     currentRank,
		TargetRank:      targetRank,
		Score:           score,
		Criteria:        criteria,
		MissingCriteria: missing,
	}, nil
}

func mapTitleToRank(titleAr string) string {
	switch titleAr {
	case "أستاذ":
		return "professor"
	case "أستاذ مشارك":
		return "associate_prof"
	case "أستاذ مساعد":
		return "assistant_prof"
	case "مدرس":
		return "lecturer"
	default:
		return "lecturer"
	}
}

func nextRank(current string) string {
	switch current {
	case "lecturer":
		return "assistant_prof"
	case "assistant_prof":
		return "associate_prof"
	case "associate_prof":
		return "professor"
	default:
		return ""
	}
}
