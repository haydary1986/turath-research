package activity

// Role names as stored in the database (kept local to avoid an import cycle with middleware).
const (
	roleAdmin             = "admin"
	roleScientificAffairs = "scientific_affairs"
	roleVPResearch        = "vp_research"
)

// Scope is one data-entry responsibility («مسؤولية إدخال»): a category, either for the
// whole university (CollegeID empty) or for one college.
type Scope struct {
	Category  string
	CollegeID string
}

// Actor is the caller as the permission rules see it. Scopes come from the database
// on every request, so a granted or withdrawn responsibility applies at once.
type Actor struct {
	UserID string
	Role   string
	Scopes []Scope
}

// IsRegistrar reports whether the role keeps the institutional registers itself.
func IsRegistrar(role string) bool {
	return role == roleAdmin || role == roleScientificAffairs
}

// SeesAllRecords reports whether the role reviews and exports every record.
func SeesAllRecords(role string) bool {
	return IsRegistrar(role) || role == roleVPResearch
}

// Holds reports whether the actor is responsible for the category at the given college.
// A nil college is a university-level record: only a university-wide scope covers it.
func (a Actor) Holds(category string, collegeID *string) bool {
	for _, s := range a.Scopes {
		if s.Category != category {
			continue
		}
		if s.CollegeID == "" || (collegeID != nil && *collegeID == s.CollegeID) {
			return true
		}
	}
	return false
}

// HoldsAny reports whether the actor carries at least one responsibility.
func (a Actor) HoldsAny() bool { return len(a.Scopes) > 0 }

// CanEnter decides whether the actor may file a record of this category at this college.
// Personal categories are open to every member for their own work; institutional ones
// need the responsibility in scope, or the scientific-affairs registrar.
func CanEnter(a Actor, category string, collegeID *string) bool {
	if !IsCategory(category) {
		return false
	}
	if !IsInstitutional(category) {
		return true
	}
	return IsRegistrar(a.Role) || a.Holds(category, collegeID)
}

// Record is the part of a stored activity the edit rules look at.
type Record struct {
	CreatedBy string
	Category  string
	CollegeID *string
	Status    string
}

// IsEditable reports whether a record's status still accepts changes.
func IsEditable(status string) bool {
	return status == StatusPending || status == StatusReturned
}

// CanEdit decides whether the actor may change an existing record (before the change is applied).
// The author edits their own; a responsibility holder edits every record of the category in scope.
func CanEdit(a Actor, rec Record) bool {
	if !IsEditable(rec.Status) {
		return false
	}
	return rec.CreatedBy == a.UserID || a.Holds(rec.Category, rec.CollegeID)
}

// CanMoveTo decides whether an edit may give the record this category and college.
// Authors follow the entry rules; a holder editing someone else's record must keep it
// inside their own responsibility, so they cannot push it somewhere they could not reach.
func CanMoveTo(a Actor, rec Record, category string, collegeID *string) bool {
	if rec.CreatedBy == a.UserID {
		return CanEnter(a, category, collegeID)
	}
	return CanEnter(a, category, collegeID) && a.Holds(category, collegeID)
}

// CanExport reports whether the actor may download the FS 003 form at all.
func CanExport(a Actor) bool {
	return SeesAllRecords(a.Role) || a.HoldsAny()
}

// ExportIncludes decides whether a record belongs in the actor's FS 003 export:
// staff export everything, holders exactly their categories within their scope.
func ExportIncludes(a Actor, category string, collegeID *string) bool {
	if SeesAllRecords(a.Role) {
		return true
	}
	return a.Holds(category, collegeID)
}
