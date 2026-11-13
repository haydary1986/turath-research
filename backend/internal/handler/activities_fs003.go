package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/haydary1986/turath-research/backend/internal/model"
	"github.com/haydary1986/turath-research/backend/internal/service/activity"
	"github.com/haydary1986/turath-research/backend/internal/service/fs003"
)

const (
	universityNameAr  = "جامعة التراث"
	maxCoordinatorLen = 200
)

// fs003Item is one candidate line of the form before scoping and formatting.
type fs003Item struct {
	category  string
	collegeID *string
	isoDate   string
	row       fs003.Row
}

// ExportFS003 fills the ministry form FS 003 with the APPROVED records of the period,
// plus the verified journal publications (sheets 4–6), for staff (everything) and for
// responsibility holders (their categories within their scope).
// Query: from, to (or year), college_id, coordinator.
func (h *PortalHandler) ExportFS003(w http.ResponseWriter, r *http.Request) {
	user := authedUser(w, r)
	if user == nil {
		return
	}
	q := r.URL.Query()
	period, problems := reportRange(q, time.Now().In(fs003.Baghdad))
	college, ok := idOrNil(q.Get("college_id"))
	if !ok {
		problems = append(problems, "college_id:invalid")
	}
	coordinator := strings.TrimSpace(q.Get("coordinator"))
	if utf8.RuneCountInString(coordinator) > maxCoordinatorLen {
		problems = append(problems, "coordinator:too_long")
	}
	if len(problems) > 0 {
		model.RespondError(w, http.StatusBadRequest, "validation:"+strings.Join(problems, ","))
		return
	}

	ctx, cancel := longRunningContext(w, r, reportTimeout)
	defer cancel()
	actor, err := loadActor(ctx, h.db, user)
	if err != nil {
		serverError(w, "Failed to load responsibilities", err)
		return
	}
	if !activity.CanExport(actor) {
		model.RespondError(w, http.StatusForbidden, "fs003_forbidden")
		return
	}

	meta, err := h.fs003Meta(ctx, user.UserID, college, coordinator, period)
	if err != nil {
		serverError(w, "Failed to build FS 003", err)
		return
	}
	items, err := h.fs003Items(ctx, period, college)
	if err != nil {
		serverError(w, "Failed to build FS 003", err)
		return
	}
	book, err := fs003.Build(meta, groupFS003(actor, items))
	if err != nil {
		serverError(w, "Failed to build FS 003", err)
		return
	}
	defer book.Close() //nolint:errcheck // in-memory workbook

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="fs003-%s.xlsx"`, period.slug))
	if err := book.Write(w); err != nil {
		// Headers are already sent; a JSON error would corrupt the file. Log only.
		slog.Error("portal: fs003 stream interrupted", "error", err)
	}
}

// fs003Meta resolves the organization line and the coordinator (the caller by default).
func (h *PortalHandler) fs003Meta(ctx context.Context, userID string, college *string, coordinator string, period reportPeriod) (fs003.Meta, error) {
	meta := fs003.Meta{
		Organization: universityNameAr,
		Period:       fs003.PeriodPhrase(period.start, period.end),
		Coordinator:  coordinator,
	}
	if college != nil {
		var name string
		err := h.db.QueryRow(ctx, `SELECT name_ar FROM colleges WHERE id = $1`, *college).Scan(&name)
		if err != nil && !isNoRows(err) {
			return meta, err
		}
		if name != "" {
			meta.Organization = universityNameAr + " – " + name
		}
	}
	if meta.Coordinator == "" {
		err := h.db.QueryRow(ctx, `SELECT COALESCE(name_ar, email) FROM users WHERE id = $1`, userID).Scan(&meta.Coordinator)
		if err != nil && !isNoRows(err) {
			return meta, err
		}
	}
	return meta, nil
}

const fs003ActivitiesQuery = `
	SELECT a.category, a.title_ar, COALESCE(a.category_detail, a.location, ''), COALESCE(a.item_count, 0),
	       to_char(a.activity_date, 'YYYY-MM-DD'), COALESCE(a.notes, ''), a.college_id::text
	FROM activities a
	WHERE a.status = 'approved' AND a.activity_date BETWEEN $1 AND $2
	  AND ($3::uuid IS NULL OR a.college_id = $3)
	ORDER BY a.activity_date, a.created_at`

// Verified journal publications, dated by publication (Baghdad day of submission as fallback).
const fs003ResearchQuery = `
	SELECT COALESCE(r.details->>'index_db', ''), COALESCE(NULLIF(r.title_en, ''), r.title_ar, ''),
	       COALESCE(r.details->>'journal_name', ''), to_char(` + reportPublicationDate + `, 'YYYY-MM-DD'),
	       COALESCE(u.name_ar, u.email), r.college_id::text
	FROM portal_requests r
	JOIN users u ON u.id = r.applicant_id
	WHERE r.kind = 'journal_verification'
	  AND r.status IN ('verified', 'awaiting_signatures', 'completed')
	  AND ` + reportPublicationDate + ` BETWEEN $1 AND $2
	  AND ($3::uuid IS NULL OR r.college_id = $3)
	ORDER BY 4, r.request_no`

// fs003Items loads every approved record and verified publication of the period.
func (h *PortalHandler) fs003Items(ctx context.Context, period reportPeriod, college *string) ([]fs003Item, error) {
	var items []fs003Item
	rows, err := h.db.Query(ctx, fs003ActivitiesQuery, period.start, period.end, college)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var it fs003Item
		if err := rows.Scan(&it.category, &it.row.Title, &it.row.Detail, &it.row.Count,
			&it.isoDate, &it.row.Notes, &it.collegeID); err != nil {
			rows.Close()
			return nil, err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = h.db.Query(ctx, fs003ResearchQuery, period.start, period.end, college)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var indexDB, journal, author string
		var it fs003Item
		if err := rows.Scan(&indexDB, &it.row.Title, &journal, &it.isoDate, &author, &it.collegeID); err != nil {
			return nil, err
		}
		if research, ok := researchItem(indexDB, journal, author, it); ok {
			items = append(items, research)
		}
	}
	return items, rows.Err()
}

// researchItem places a verified publication on sheet 4 (journal name), 5 or 6 (index database).
func researchItem(indexDB, journal, author string, it fs003Item) (fs003Item, bool) {
	it.category = activity.CategoryForIndexDB(indexDB)
	if it.category == "" {
		return it, false
	}
	if it.category == activity.CategoryLocalResearch {
		it.row.Detail = journal
	} else {
		it.row.Detail = researchIndexLabel(indexDB)
	}
	it.row.Notes = "الباحث: " + author
	return it, true
}

func researchIndexLabel(indexDB string) string {
	switch indexDB {
	case "scopus":
		return "Scopus"
	case "wos":
		return "Web of Science"
	case "international_unindexed":
		return "عالمية غير مستوعبة"
	}
	return indexDB
}

// groupFS003 keeps what the actor may export, orders each sheet by date and formats the dates.
func groupFS003(actor activity.Actor, items []fs003Item) map[string][]fs003.Row {
	kept := make([]fs003Item, 0, len(items))
	for _, it := range items {
		if activity.IsCategory(it.category) && activity.ExportIncludes(actor, it.category, it.collegeID) {
			kept = append(kept, it)
		}
	}
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].isoDate < kept[j].isoDate })
	out := map[string][]fs003.Row{}
	for _, it := range kept {
		row := it.row
		row.Date = fs003.FormatDate(it.isoDate)
		out[it.category] = append(out[it.category], row)
	}
	return out
}
