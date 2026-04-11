package incentive

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Calculator computes publication incentive amounts based on institutional rules.
type Calculator struct {
	db *pgxpool.Pool
}

func NewCalculator(db *pgxpool.Pool) *Calculator {
	return &Calculator{db: db}
}

// IncentiveCalcResult holds the calculated incentive for a publication.
type IncentiveCalcResult struct {
	PublicationID        string  `json:"publication_id"`
	JournalName          string  `json:"journal_name"`
	JournalClassification string `json:"journal_classification"`
	BaseAmount           float64 `json:"base_amount"`
	CorrespondingBonus   float64 `json:"corresponding_bonus"`
	TotalAmount          float64 `json:"total_amount"`
	AuthorShare          float64 `json:"author_share_pct"`
	FinalAmount          float64 `json:"final_amount"`
	IsCorresponding      bool    `json:"is_corresponding"`
	TotalAuthors         int     `json:"total_authors"`
}

// CalculateForResearcher computes incentives for all eligible publications of a researcher.
func (c *Calculator) CalculateForResearcher(ctx context.Context, researcherID, institutionID string) ([]IncentiveCalcResult, error) {
	rows, err := c.db.Query(ctx, `
		SELECT p.id, COALESCE(j.name, ''), COALESCE(j.sjr_ranking::text, 'unranked'),
		       pa.is_corresponding,
		       (SELECT COUNT(*) FROM publication_authors WHERE publication_id = p.id) AS total_authors
		FROM publications p
		JOIN publication_authors pa ON pa.publication_id = p.id
		LEFT JOIN journals j ON j.id = p.journal_id
		WHERE pa.researcher_id = $1
		  AND NOT EXISTS (
		      SELECT 1 FROM publication_incentives pi
		      WHERE pi.publication_id = p.id AND pi.researcher_id = $1
		  )
		ORDER BY p.publication_date DESC
	`, researcherID)
	if err != nil {
		return nil, fmt.Errorf("query publications: %w", err)
	}
	defer rows.Close()

	var results []IncentiveCalcResult
	for rows.Next() {
		var r IncentiveCalcResult
		rows.Scan(&r.PublicationID, &r.JournalName, &r.JournalClassification, &r.IsCorresponding, &r.TotalAuthors)

		// Map SJR ranking to classification
		classification := mapRankingToClassification(r.JournalClassification)

		// Get incentive rule
		var baseAmount, corrBonus float64
		var maxAuthors int
		err := c.db.QueryRow(ctx, `
			SELECT base_amount, corresponding_bonus, max_authors_share
			FROM incentive_rules
			WHERE institution_id = $1 AND journal_classification = $2::journal_classification
			  AND effective_from <= CURRENT_DATE
			  AND (effective_to IS NULL OR effective_to >= CURRENT_DATE)
			ORDER BY effective_from DESC LIMIT 1
		`, institutionID, classification).Scan(&baseAmount, &corrBonus, &maxAuthors)

		if err != nil {
			continue // No rule for this classification
		}

		r.BaseAmount = baseAmount
		r.CorrespondingBonus = 0
		if r.IsCorresponding {
			r.CorrespondingBonus = corrBonus
		}
		r.TotalAmount = r.BaseAmount + r.CorrespondingBonus

		// Calculate author share
		shareAuthors := r.TotalAuthors
		if maxAuthors > 0 && shareAuthors > maxAuthors {
			shareAuthors = maxAuthors
		}
		if shareAuthors < 1 {
			shareAuthors = 1
		}
		r.AuthorShare = 100.0 / float64(shareAuthors)
		r.FinalAmount = r.TotalAmount * r.AuthorShare / 100.0

		results = append(results, r)
	}

	return results, nil
}

func mapRankingToClassification(ranking string) string {
	switch ranking {
	case "Q1":
		return "Q1"
	case "Q2":
		return "Q2"
	case "Q3":
		return "Q3"
	case "Q4":
		return "Q4"
	default:
		return "scopus_only"
	}
}
