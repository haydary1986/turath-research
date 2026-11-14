package rissync

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// change is one row of the run's change log.
type change struct {
	runID, researcherID, field string
	action                     Action
	old, new, note             string
	applied                    bool
}

func loadState(ctx context.Context, db *pgxpool.Pool) (map[string]map[string]string, error) {
	rows, err := db.Query(ctx, `SELECT researcher_id::text, field, value FROM ris_sync_state`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]map[string]string{}
	for rows.Next() {
		var id, field, value string
		if err := rows.Scan(&id, &field, &value); err != nil {
			return nil, err
		}
		if out[id] == nil {
			out[id] = map[string]string{}
		}
		out[id][field] = value
	}
	return out, rows.Err()
}

func loadOpenReviews(ctx context.Context, db *pgxpool.Pool) (map[string]bool, error) {
	rows, err := db.Query(ctx, `SELECT researcher_id::text, field FROM ris_sync_reviews WHERE resolved_at IS NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id, field string
		if err := rows.Scan(&id, &field); err != nil {
			return nil, err
		}
		out[id+"|"+field] = true
	}
	return out, rows.Err()
}

func linkRecords(ctx context.Context, db *pgxpool.Pool, researcherID, risID string) error {
	_, err := db.Exec(ctx, `
		INSERT INTO ris_sync_links (researcher_id, ris_researcher_id) VALUES ($1, $2)
		ON CONFLICT (researcher_id) DO UPDATE SET ris_researcher_id = EXCLUDED.ris_researcher_id
		WHERE ris_sync_links.ris_researcher_id <> EXCLUDED.ris_researcher_id`, researcherID, risID)
	return err
}

func (e *Engine) saveState(ctx context.Context, researcherID, field, value string) {
	if _, err := e.db.Exec(ctx, `
		INSERT INTO ris_sync_state (researcher_id, field, value) VALUES ($1, $2, $3)
		ON CONFLICT (researcher_id, field) DO UPDATE SET value = EXCLUDED.value, synced_at = NOW()`,
		researcherID, field, value); err != nil {
		slog.Error("ris sync: failed to save state", "field", field, "error", err)
	}
}

func (e *Engine) openReview(ctx context.Context, researcherID, field, portalValue, risValue, reason string) {
	if _, err := e.db.Exec(ctx, `
		INSERT INTO ris_sync_reviews (researcher_id, field, portal_value, ris_value, reason)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (researcher_id, field) WHERE resolved_at IS NULL DO NOTHING`,
		researcherID, field, portalValue, risValue, reason); err != nil {
		slog.Error("ris sync: failed to open a review", "field", field, "error", err)
	}
}

func (e *Engine) logChange(ctx context.Context, c change) {
	if _, err := e.db.Exec(ctx, `
		INSERT INTO ris_sync_changes (run_id, researcher_id, field, action, old_value, new_value, applied, note)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, NULLIF($8, ''))`,
		c.runID, c.researcherID, c.field, c.action.String(), c.old, c.new, c.applied, c.note); err != nil {
		slog.Error("ris sync: failed to log a change", "field", c.field, "error", err)
	}
}

// Review resolution choices.
const (
	UsePortal = "portal"
	UseRIS    = "ris"
)

// ErrReviewNotFound means the review does not exist or is already resolved.
var ErrReviewNotFound = errors.New("review not found or already resolved")

// Resolve applies an admin's choice for an open review: the chosen side's value is copied
// to the other side, recorded as agreed, and the field syncs normally again.
func (e *Engine) Resolve(ctx context.Context, reviewID, use, actorID string) error {
	var researcherID, field, portalValue, risValue string
	err := e.db.QueryRow(ctx, `SELECT researcher_id::text, field, portal_value, ris_value FROM ris_sync_reviews
		WHERE id = $1 AND resolved_at IS NULL`, reviewID).Scan(&researcherID, &field, &portalValue, &risValue)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrReviewNotFound
	}
	if err != nil {
		return err
	}
	value := portalValue
	if use == UseRIS {
		value = risValue
		if err := applyToPortal(ctx, e.db, researcherID, field, value); err != nil {
			return err
		}
	} else if err := e.pushValue(ctx, researcherID, field, value); err != nil {
		return err
	}
	e.saveState(ctx, researcherID, field, value)
	_, err = e.db.Exec(ctx, `UPDATE ris_sync_reviews SET resolved_at = NOW(), resolved_by = NULLIF($2, '')::uuid, resolution = $3
		WHERE id = $1`, reviewID, actorID, use)
	return err
}

// pushValue writes one portal value into the linked RIS researcher.
func (e *Engine) pushValue(ctx context.Context, researcherID, field, value string) error {
	var risID string
	if err := e.db.QueryRow(ctx, `SELECT ris_researcher_id::text FROM ris_sync_links WHERE researcher_id = $1`,
		researcherID).Scan(&risID); err != nil {
		return err
	}
	var risDept Org
	if field == FieldDepartment && value != "" {
		rc, depts, err := e.ris.Orgs(ctx)
		if err != nil {
			return err
		}
		pc, pd, err := loadPortalOrgs(ctx, e.db)
		if err != nil {
			return err
		}
		if id, ok := BuildDepartmentMap(pc, pd, rc, depts).ToRIS(value); ok {
			for _, d := range depts {
				if d.ID == id {
					risDept = d
				}
			}
		}
	}
	columns, platform, skip := risUpdate(field, value, risDept)
	if skip != "" {
		return errors.New("cannot write to RIS: " + skip)
	}
	if platform != "" {
		return e.ris.SetProfile(ctx, risID, platform, RISLinkURL(field, value))
	}
	return e.ris.UpdateResearcher(ctx, risID, columns)
}
