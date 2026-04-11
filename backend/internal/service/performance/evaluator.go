package performance

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Evaluator generates annual performance reviews for researchers.
type Evaluator struct {
	db *pgxpool.Pool
}

func NewEvaluator(db *pgxpool.Pool) *Evaluator {
	return &Evaluator{db: db}
}

// ReviewResult is the computed performance review for a researcher.
type ReviewResult struct {
	ResearcherID       string  `json:"researcher_id"`
	ResearcherName     string  `json:"researcher_name"`
	ReviewYear         int     `json:"review_year"`
	Q1Publications     int     `json:"q1_publications"`
	Q2Publications     int     `json:"q2_publications"`
	TotalPublications  int     `json:"total_publications"`
	CitationsThisYear  int     `json:"citations_this_year"`
	HIndexCurrent      int     `json:"h_index_current"`
	ConferencesAttended int    `json:"conferences_attended"`
	WeightedScore      float64 `json:"weighted_score"`
	RankInDepartment   int     `json:"rank_in_department"`
}

// GenerateForDepartment generates performance reviews for all researchers in a department.
func (e *Evaluator) GenerateForDepartment(ctx context.Context, departmentID string, year int, institutionID string) ([]ReviewResult, error) {
	rows, err := e.db.Query(ctx, `
		SELECT r.id, r.name_ar, r.h_index, r.total_citations, r.total_publications
		FROM researchers r
		WHERE r.department_id = $1 AND r.status = 'active'
		ORDER BY r.name_ar
	`, departmentID)
	if err != nil {
		return nil, fmt.Errorf("fetch researchers: %w", err)
	}
	defer rows.Close()

	var reviews []ReviewResult
	for rows.Next() {
		var rID, name string
		var hIndex, totalCit, totalPub int
		rows.Scan(&rID, &name, &hIndex, &totalCit, &totalPub)

		// Count Q1/Q2 pubs for this year
		var q1, q2, yearPubs int
		e.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM publications p
			JOIN publication_authors pa ON pa.publication_id = p.id
			JOIN journals j ON j.id = p.journal_id
			WHERE pa.researcher_id = $1 AND j.sjr_ranking = 'Q1'
			  AND EXTRACT(YEAR FROM p.publication_date) = $2
		`, rID, year).Scan(&q1)
		e.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM publications p
			JOIN publication_authors pa ON pa.publication_id = p.id
			JOIN journals j ON j.id = p.journal_id
			WHERE pa.researcher_id = $1 AND j.sjr_ranking = 'Q2'
			  AND EXTRACT(YEAR FROM p.publication_date) = $2
		`, rID, year).Scan(&q2)
		e.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM publications p
			JOIN publication_authors pa ON pa.publication_id = p.id
			WHERE pa.researcher_id = $1
			  AND EXTRACT(YEAR FROM p.publication_date) = $2
		`, rID, year).Scan(&yearPubs)

		var confCount int
		e.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM conference_participations
			WHERE researcher_id = $1 AND EXTRACT(YEAR FROM conference_date_start) = $2
		`, rID, year).Scan(&confCount)

		// Weighted score: Q1*10 + Q2*7 + other*3 + h_index*2 + conferences*1
		otherPubs := yearPubs - q1 - q2
		if otherPubs < 0 {
			otherPubs = 0
		}
		score := float64(q1)*10 + float64(q2)*7 + float64(otherPubs)*3 + float64(hIndex)*2 + float64(confCount)*1

		reviews = append(reviews, ReviewResult{
			ResearcherID:       rID,
			ResearcherName:     name,
			ReviewYear:         year,
			Q1Publications:     q1,
			Q2Publications:     q2,
			TotalPublications:  yearPubs,
			HIndexCurrent:      hIndex,
			ConferencesAttended: confCount,
			WeightedScore:      score,
		})
	}

	// Rank within department
	for i := range reviews {
		rank := 1
		for j := range reviews {
			if reviews[j].WeightedScore > reviews[i].WeightedScore {
				rank++
			}
		}
		reviews[i].RankInDepartment = rank
	}

	// Persist reviews
	for _, r := range reviews {
		e.db.Exec(ctx, `
			INSERT INTO annual_performance_reviews
				(id, researcher_id, review_year, q1_publications, q2_publications,
				 total_publications, h_index_current, conferences_attended,
				 weighted_score, rank_in_department, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'auto_generated')
			ON CONFLICT (researcher_id, review_year) DO UPDATE SET
				q1_publications = EXCLUDED.q1_publications,
				q2_publications = EXCLUDED.q2_publications,
				total_publications = EXCLUDED.total_publications,
				h_index_current = EXCLUDED.h_index_current,
				conferences_attended = EXCLUDED.conferences_attended,
				weighted_score = EXCLUDED.weighted_score,
				rank_in_department = EXCLUDED.rank_in_department,
				updated_at = NOW()
		`, uuid.New().String(), r.ResearcherID, r.ReviewYear,
			r.Q1Publications, r.Q2Publications, r.TotalPublications,
			r.HIndexCurrent, r.ConferencesAttended, r.WeightedScore, r.RankInDepartment)
	}

	return reviews, nil
}
