package activity

import "testing"

func strp(s string) *string { return &s }

const (
	collegeA = "college-a"
	collegeB = "college-b"
)

func researcher(scopes ...Scope) Actor {
	return Actor{UserID: "u1", Role: "researcher", Scopes: scopes}
}

func TestCanEnterPersonalCategories(t *testing.T) {
	for _, c := range []string{CategoryWorkshops, CategoryCourses, CategoryBooks, CategoryPatents,
		CategoryConferences, CategoryFellowships, CategoryCommunityService,
		CategoryLocalResearch, CategoryIndexedResearch, CategoryInternationalResearch} {
		if !CanEnter(researcher(), c, strp(collegeA)) {
			t.Errorf("any member enters their own %s", c)
		}
	}
	if CanEnter(researcher(), "unknown", strp(collegeA)) {
		t.Error("unknown category accepted")
	}
}

func TestCanEnterInstitutionalCategories(t *testing.T) {
	for _, c := range []string{CategoryInfrastructure, CategoryInternationalAgreements, CategoryLocalAgreements} {
		if CanEnter(researcher(), c, strp(collegeA)) {
			t.Errorf("plain researcher entered %s", c)
		}
		if !CanEnter(Actor{UserID: "s", Role: "scientific_affairs"}, c, nil) {
			t.Errorf("scientific affairs enters %s", c)
		}
		if !CanEnter(Actor{UserID: "a", Role: "admin"}, c, strp(collegeA)) {
			t.Errorf("admin enters %s", c)
		}
		if CanEnter(Actor{UserID: "v", Role: "vp_research"}, c, nil) {
			t.Errorf("VP is a reviewer, not a registrar, for %s", c)
		}
	}
}

func TestResponsibilityScope(t *testing.T) {
	university := researcher(Scope{Category: CategoryInternationalAgreements})
	college := researcher(Scope{Category: CategoryInternationalAgreements, CollegeID: collegeA})

	if !CanEnter(university, CategoryInternationalAgreements, nil) ||
		!CanEnter(university, CategoryInternationalAgreements, strp(collegeB)) {
		t.Error("university-wide holder covers every college and university-level records")
	}
	if CanEnter(university, CategoryLocalAgreements, nil) {
		t.Error("a responsibility covers only its own category")
	}
	if !CanEnter(college, CategoryInternationalAgreements, strp(collegeA)) {
		t.Error("college holder enters for that college")
	}
	if CanEnter(college, CategoryInternationalAgreements, strp(collegeB)) {
		t.Error("college holder cannot enter for another college")
	}
	if CanEnter(college, CategoryInternationalAgreements, nil) {
		t.Error("college holder cannot enter university-level records")
	}
}

func TestCanEdit(t *testing.T) {
	holder := Actor{UserID: "h", Role: "researcher", Scopes: []Scope{{Category: CategoryLocalAgreements, CollegeID: collegeA}}}
	theirs := Record{CreatedBy: "someone", Category: CategoryLocalAgreements, CollegeID: strp(collegeA), Status: StatusPending}

	if !CanEdit(holder, theirs) {
		t.Error("holder edits others' records in scope")
	}
	if CanEdit(holder, Record{CreatedBy: "someone", Category: CategoryLocalAgreements, CollegeID: strp(collegeB), Status: StatusPending}) {
		t.Error("holder cannot edit another college's record")
	}
	if CanEdit(holder, Record{CreatedBy: "someone", Category: CategoryBooks, CollegeID: strp(collegeA), Status: StatusPending}) {
		t.Error("holder cannot edit a category they do not hold")
	}
	approved := theirs
	approved.Status = StatusApproved
	if CanEdit(holder, approved) || CanEdit(Actor{UserID: "someone"}, approved) {
		t.Error("decided records are final")
	}
	if !CanEdit(Actor{UserID: "someone", Role: "researcher"}, theirs) {
		t.Error("the author edits their own pending record")
	}
	if CanEdit(Actor{UserID: "x", Role: "scientific_affairs"}, theirs) {
		t.Error("reviewers return records instead of editing them")
	}

	if !CanMoveTo(holder, theirs, CategoryLocalAgreements, strp(collegeA)) {
		t.Error("holder keeps the record in scope")
	}
	if CanMoveTo(holder, theirs, CategoryLocalAgreements, strp(collegeB)) {
		t.Error("holder cannot move a record out of their scope")
	}
	if CanMoveTo(holder, theirs, CategoryWorkshops, strp(collegeA)) {
		t.Error("holder cannot recategorize into a category they do not hold")
	}
	author := Actor{UserID: "someone", Role: "researcher"}
	if !CanMoveTo(author, theirs, CategoryWorkshops, strp(collegeA)) {
		t.Error("the author may recategorize into a personal category")
	}
	if CanMoveTo(author, theirs, CategoryInfrastructure, strp(collegeA)) {
		t.Error("the author still needs the responsibility for institutional categories")
	}
}

func TestExportRules(t *testing.T) {
	if CanExport(researcher()) {
		t.Error("plain researcher cannot export")
	}
	for _, role := range []string{"admin", "scientific_affairs", "vp_research"} {
		a := Actor{UserID: "s", Role: role}
		if !CanExport(a) || !ExportIncludes(a, CategoryBooks, nil) {
			t.Errorf("%s exports everything", role)
		}
	}
	holder := researcher(Scope{Category: CategoryInternationalAgreements, CollegeID: collegeA})
	if !CanExport(holder) {
		t.Error("holder exports their scope")
	}
	if !ExportIncludes(holder, CategoryInternationalAgreements, strp(collegeA)) {
		t.Error("holder's category in scope is exported")
	}
	if ExportIncludes(holder, CategoryInternationalAgreements, strp(collegeB)) ||
		ExportIncludes(holder, CategoryBooks, strp(collegeA)) {
		t.Error("holder export leaks outside the scope")
	}
}
