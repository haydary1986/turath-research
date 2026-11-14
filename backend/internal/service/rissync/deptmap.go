package rissync

import (
	"regexp"
	"strings"
)

// Org is a college or department as either system names it.
type Org struct {
	ID        string
	NameAr    string
	CollegeID string // departments only
}

var (
	arabicMarks  = regexp.MustCompile(`[\x{064B}-\x{0652}\x{0640}]`)
	orgPrefix    = regexp.MustCompile(`^(كليه|قسم)\s+`)
	orgSuffix    = regexp.MustCompile(`\s*[-–—(].*$`) // "اللغة الإنجليزية - آداب" → "اللغة الإنجليزية"
	hamzaAlef    = strings.NewReplacer("أ", "ا", "إ", "ا", "آ", "ا", "ة", "ه", "ى", "ي")
	whitespaceRe = regexp.MustCompile(`\s+`)
)

// NormalizeOrgName reduces an Arabic college or department name to a matching key:
// hamza and taa marbuta forms, diacritics, the "كلية/قسم" prefix and a trailing
// qualifier in brackets or after a dash are ignored.
func NormalizeOrgName(name string) string {
	s := hamzaAlef.Replace(strings.TrimSpace(name))
	s = arabicMarks.ReplaceAllString(s, "")
	s = whitespaceRe.ReplaceAllString(s, " ")
	s = orgPrefix.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// DepartmentMap translates department ids between the two systems. A department is
// matched by its normalized name within the same-named college; a name with no single
// match stays unmapped and its researchers' department is not synced.
type DepartmentMap struct {
	toRIS    map[string]string // portal department id → RIS department id
	toPortal map[string]string
	// UnmappedRIS lists RIS department names with no portal counterpart.
	UnmappedRIS []string
}

// BuildDepartmentMap matches the two department lists (colleges first, then departments).
func BuildDepartmentMap(portalColleges, portalDepts, risColleges, risDepts []Org) DepartmentMap {
	collegeKey := func(colleges []Org) map[string]string {
		out := map[string]string{}
		for _, c := range colleges {
			out[c.ID] = NormalizeOrgName(c.NameAr)
		}
		return out
	}
	pCollege, rCollege := collegeKey(portalColleges), collegeKey(risColleges)

	// key = college key + department key, tried with and without the trailing qualifier.
	index := map[string][]string{}
	for _, d := range portalDepts {
		for _, k := range deptKeys(pCollege[d.CollegeID], d.NameAr) {
			index[k] = append(index[k], d.ID)
		}
	}
	m := DepartmentMap{toRIS: map[string]string{}, toPortal: map[string]string{}}
	for _, d := range risDepts {
		matched := ""
		for _, k := range deptKeys(rCollege[d.CollegeID], d.NameAr) {
			if ids := unique(index[k]); len(ids) == 1 {
				matched = ids[0]
				break
			}
		}
		if matched == "" {
			m.UnmappedRIS = append(m.UnmappedRIS, d.NameAr)
			continue
		}
		if _, taken := m.toRIS[matched]; taken {
			continue // two RIS departments for one portal department: keep the first
		}
		m.toRIS[matched], m.toPortal[d.ID] = d.ID, matched
	}
	return m
}

func deptKeys(college, name string) []string {
	full := NormalizeOrgName(name)
	short := strings.TrimSpace(orgSuffix.ReplaceAllString(full, ""))
	keys := []string{college + "|" + full}
	if short != full && short != "" {
		keys = append(keys, college+"|"+short)
	}
	// A department renamed between colleges is still found by its name alone.
	return append(keys, "|"+full)
}

func unique(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// ToRIS returns the RIS department for a portal department.
func (m DepartmentMap) ToRIS(portalID string) (string, bool) {
	id, ok := m.toRIS[portalID]
	return id, ok
}

// ToPortal returns the portal department for a RIS department.
func (m DepartmentMap) ToPortal(risID string) (string, bool) {
	id, ok := m.toPortal[risID]
	return id, ok
}
