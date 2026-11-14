package rissync

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Run modes and triggers.
const (
	ModeDryRun      = "dry_run"
	ModeApply       = "apply"
	TriggerSchedule = "schedule"
	TriggerManual   = "manual"
)

// advisoryLockKey keeps two sync runs (two instances, or a manual run during a
// scheduled one) from working at the same time.
const advisoryLockKey = 7_337_001

// ErrBusy means another sync run holds the lock.
var ErrBusy = errors.New("a RIS sync run is already in progress")

// Engine runs the sync between the portal database and the RIS.
type Engine struct {
	db  *pgxpool.Pool
	ris *Client
}

// NewEngine returns nil when the RIS is not configured.
func NewEngine(db *pgxpool.Pool, ris *Client) *Engine {
	if db == nil || ris == nil {
		return nil
	}
	return &Engine{db: db, ris: ris}
}

// Stats summarises one run.
type Stats struct {
	RISResearchers      int      `json:"ris_researchers"`
	Matched             int      `json:"matched"`
	UnmatchedRIS        int      `json:"unmatched_ris"`
	ToRIS               int      `json:"to_ris"`
	ToPortal            int      `json:"to_portal"`
	Agreed              int      `json:"agreed"`
	Reviews             int      `json:"reviews"`
	Skipped             int      `json:"skipped"`
	Failed              int      `json:"failed"`
	UnmappedDepartments []string `json:"unmapped_departments"`
}

// snapshot is everything one run reads before deciding.
type snapshot struct {
	ris      []RISResearcher
	emails   map[string]string
	portal   map[string]PortalResearcher
	depts    DepartmentMap
	risDepts map[string]Org
	state    map[string]map[string]string
	open     map[string]bool // researcherID|field with an open review
}

// Run performs one sync. In dry-run mode it only records what it would do.
func (e *Engine) Run(ctx context.Context, mode, trigger, actorID string) (string, Stats, error) {
	return e.runNotify(ctx, mode, trigger, actorID, nil)
}

// runNotify is Run that reports the run id on started as soon as the run row exists.
func (e *Engine) runNotify(ctx context.Context, mode, trigger, actorID string, started chan<- string) (string, Stats, error) {
	conn, err := e.db.Acquire(ctx)
	if err != nil {
		return "", Stats{}, err
	}
	defer conn.Release()
	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, advisoryLockKey).Scan(&locked); err != nil {
		return "", Stats{}, err
	}
	if !locked {
		return "", Stats{}, ErrBusy
	}
	defer conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, advisoryLockKey) //nolint:errcheck // released with the session anyway

	var runID string
	if err := e.db.QueryRow(ctx, `INSERT INTO ris_sync_runs (mode, trigger, started_by) VALUES ($1, $2, NULLIF($3, '')::uuid) RETURNING id`,
		mode, trigger, actorID).Scan(&runID); err != nil {
		return "", Stats{}, err
	}
	if started != nil {
		started <- runID
	}

	stats, runErr := e.run(ctx, runID, mode == ModeApply)
	statsJSON, _ := json.Marshal(stats)
	errText := ""
	if runErr != nil {
		errText = runErr.Error()
	}
	if _, err := e.db.Exec(context.WithoutCancel(ctx),
		`UPDATE ris_sync_runs SET finished_at = NOW(), stats = $2, error = NULLIF($3, '') WHERE id = $1`,
		runID, statsJSON, errText); err != nil {
		slog.Error("ris sync: failed to close the run", "run_id", runID, "error", err)
	}
	return runID, stats, runErr
}

func (e *Engine) load(ctx context.Context) (snapshot, error) {
	var s snapshot
	var err error
	if s.ris, err = e.ris.Researchers(ctx); err != nil {
		return s, err
	}
	if s.emails, err = e.ris.UserEmails(ctx); err != nil {
		return s, err
	}
	risColleges, risDepts, err := e.ris.Orgs(ctx)
	if err != nil {
		return s, err
	}
	if s.portal, err = loadPortal(ctx, e.db); err != nil {
		return s, err
	}
	portalColleges, portalDepts, err := loadPortalOrgs(ctx, e.db)
	if err != nil {
		return s, err
	}
	s.depts = BuildDepartmentMap(portalColleges, portalDepts, risColleges, risDepts)
	s.risDepts = map[string]Org{}
	for _, d := range risDepts {
		s.risDepts[d.ID] = d
	}
	if s.state, err = loadState(ctx, e.db); err != nil {
		return s, err
	}
	s.open, err = loadOpenReviews(ctx, e.db)
	return s, err
}

func (e *Engine) run(ctx context.Context, runID string, apply bool) (Stats, error) {
	s, err := e.load(ctx)
	if err != nil {
		return Stats{}, err
	}
	st := Stats{RISResearchers: len(s.ris), UnmappedDepartments: s.depts.UnmappedRIS}
	for _, r := range s.ris {
		p, ok := s.portal[risEmail(r, s.emails)]
		if !ok {
			st.UnmatchedRIS++
			continue
		}
		st.Matched++
		if apply {
			if err := linkRecords(ctx, e.db, p.ID, r.ID); err != nil {
				return st, err
			}
		}
		for _, spec := range Fields {
			if s.open[p.ID+"|"+spec.Key] {
				continue // an admin decides this field first
			}
			e.syncField(ctx, runID, apply, s, p, r, spec, &st)
		}
	}
	return st, nil
}

func risEmail(r RISResearcher, emails map[string]string) string {
	if r.UserID != nil {
		if email := emails[*r.UserID]; email != "" {
			return email
		}
	}
	if r.PrivateEmail != nil {
		return strings.ToLower(strings.TrimSpace(*r.PrivateEmail))
	}
	return ""
}

// risValue returns the RIS side of a field in portal terms and when RIS last changed it.
// valid is false for a value the portal cannot accept (a malformed link, an unmatched department).
func risValue(s snapshot, r RISResearcher, field string) (value string, changed time.Time, valid bool) {
	switch field {
	case FieldNameAr:
		return CanonicalName(r.FullNameAr), r.UpdatedAt, true
	case FieldNameEn:
		return CanonicalName(r.FullNameEn), r.UpdatedAt, true
	case FieldPhone:
		raw := deref(r.PrivatePhone)
		v := CanonicalPhone(raw)
		return v, r.UpdatedAt, v != "" || strings.TrimSpace(raw) == ""
	case FieldDepartment:
		if r.DepartmentID == nil {
			return "", r.UpdatedAt, true
		}
		id, ok := s.depts.ToPortal(*r.DepartmentID)
		return id, r.UpdatedAt, ok
	}
	for _, prof := range r.Profiles {
		if prof.Platform == field {
			v, ok := CanonicalLink(field, prof.URL)
			return v, prof.UpdatedAt, ok
		}
	}
	return "", r.UpdatedAt, true
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (e *Engine) syncField(ctx context.Context, runID string, apply bool, s snapshot, p PortalResearcher, r RISResearcher, spec FieldSpec, st *Stats) {
	last, hasLast := s.state[p.ID][spec.Key]
	risVal, risChanged, valid := risValue(s, r, spec.Key)
	if !valid {
		// An unusable RIS value is never copied; until it is fixed RIS counts as unchanged.
		risVal = last
		if !hasLast {
			risVal = p.Values[spec.Key]
		}
	}
	v := Values{Portal: p.Values[spec.Key], RIS: risVal, Last: last, HasLast: hasLast, PortalNewer: p.UpdatedAt.After(risChanged)}
	action := Decide(v, spec.Policy)
	ch := change{runID: runID, researcherID: p.ID, field: spec.Key, action: action}

	switch action {
	case Nothing:
		return
	case Agree:
		st.Agreed++
		if apply {
			e.saveState(ctx, p.ID, spec.Key, v.Portal)
		}
		return
	case Review:
		st.Reviews++
		ch.old, ch.new = v.RIS, v.Portal
		if apply {
			e.openReview(ctx, p.ID, spec.Key, v.Portal, v.RIS, "conflict")
		}
	case ToRIS:
		ch.old, ch.new = v.RIS, v.Portal
		e.pushToRIS(ctx, apply, s, r, spec.Key, v.Portal, &ch, st)
	case ToPortal:
		ch.old, ch.new = v.Portal, v.RIS
		e.pullToPortal(ctx, apply, p, spec.Key, v, &ch, st)
	}
	e.logChange(ctx, ch)
}

func (e *Engine) pushToRIS(ctx context.Context, apply bool, s snapshot, r RISResearcher, field, value string, ch *change, st *Stats) {
	var risDept Org
	if field == FieldDepartment && value != "" {
		if id, ok := s.depts.ToRIS(value); ok {
			risDept = s.risDepts[id]
		}
	}
	columns, platform, skip := risUpdate(field, value, risDept)
	if skip != "" {
		st.Skipped++
		ch.note = skip
		return
	}
	st.ToRIS++
	if !apply {
		return
	}
	var err error
	if platform != "" {
		err = e.ris.SetProfile(ctx, r.ID, platform, RISLinkURL(field, value))
	} else {
		err = e.ris.UpdateResearcher(ctx, r.ID, columns)
	}
	if err != nil {
		st.Failed++
		ch.note = "failed"
		slog.Error("ris sync: write to RIS failed", "field", field, "error", err)
		return
	}
	ch.applied = true
	e.saveState(ctx, ch.researcherID, field, value)
}

func (e *Engine) pullToPortal(ctx context.Context, apply bool, p PortalResearcher, field string, v Values, ch *change, st *Stats) {
	st.ToPortal++
	if !apply {
		return
	}
	err := applyToPortal(ctx, e.db, p.ID, field, v.RIS)
	switch {
	case errors.Is(err, errTaken):
		// The id is already another researcher's profile: a person must decide.
		st.ToPortal--
		st.Reviews++
		ch.action, ch.note = Review, "taken"
		e.openReview(ctx, p.ID, field, v.Portal, v.RIS, "taken")
	case err != nil:
		st.Failed++
		ch.note = "failed"
		slog.Error("ris sync: write to portal failed", "field", field, "error", err)
	default:
		ch.applied = true
		e.saveState(ctx, p.ID, field, v.RIS)
	}
}
