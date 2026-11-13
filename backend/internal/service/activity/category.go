package activity

// Categories of the ministry form FS 003 «استمارة تقدم عمل الجامعة او الهيئة».
// The keys are stable identifiers stored in the database; the order is the
// order of the form's sheets (the form has no sheet 8).
const (
	CategoryInfrastructure          = "infrastructure"           // 1  البنى التحتية
	CategoryWorkshops               = "workshops"                // 2  ورش وندوات وحلقات نقاشية
	CategoryCourses                 = "courses"                  // 3  الدورات
	CategoryLocalResearch           = "local_research"           // 4  البحوث المنشورة محليا
	CategoryIndexedResearch         = "indexed_research"         // 5  المستوعبات العالمية
	CategoryInternationalResearch   = "international_research"   // 6  البحوث المنشورة دوليا
	CategoryBooks                   = "books"                    // 7  الكتب المؤلفة والمترجمة
	CategoryPatents                 = "patents"                  // 9  براءات الاختراع
	CategoryConferences             = "conferences"              // 10 المؤتمرات العلمية
	CategoryInternationalAgreements = "international_agreements" // 11 الاتفاقيات الدولية
	CategoryLocalAgreements         = "local_agreements"         // 12 الاتفاقيات المحلية
	CategoryFellowships             = "fellowships"              // 13 الزمالات البحثية وما بعد الدكتوراه والتفرغ العلمي
	CategoryCommunityService        = "community_service"        // 14 خدمة المجتمع
)

// Categories lists every category in the form's sheet order.
var Categories = []string{
	CategoryInfrastructure, CategoryWorkshops, CategoryCourses,
	CategoryLocalResearch, CategoryIndexedResearch, CategoryInternationalResearch,
	CategoryBooks, CategoryPatents, CategoryConferences,
	CategoryInternationalAgreements, CategoryLocalAgreements,
	CategoryFellowships, CategoryCommunityService,
}

var categorySet = func() map[string]struct{} {
	m := make(map[string]struct{}, len(Categories))
	for _, c := range Categories {
		m[c] = struct{}{}
	}
	return m
}()

// IsCategory reports whether c is a known FS 003 category.
func IsCategory(c string) bool {
	_, ok := categorySet[c]
	return ok
}

// Institutional categories describe the university, not one member's work: only a
// holder of the matching responsibility or scientific affairs may enter them.
var institutional = map[string]struct{}{
	CategoryInfrastructure:          {},
	CategoryInternationalAgreements: {},
	CategoryLocalAgreements:         {},
}

// IsInstitutional reports whether the category is entered on behalf of the institution.
func IsInstitutional(c string) bool {
	_, ok := institutional[c]
	return ok
}

// Place categories use the form column «المكان»; it mirrors the activity's location.
var placeCategories = map[string]struct{}{
	CategoryWorkshops: {}, CategoryCourses: {}, CategoryConferences: {}, CategoryCommunityService: {},
}

// IsPlaceCategory reports whether the category's form column is the venue.
func IsPlaceCategory(c string) bool {
	_, ok := placeCategories[c]
	return ok
}

// detailRequired names the categories whose form column has no sensible blank:
// a book without a publisher or an agreement without a counter-party.
var detailRequired = map[string]struct{}{
	CategoryLocalResearch: {}, CategoryIndexedResearch: {}, CategoryInternationalResearch: {},
	CategoryBooks: {}, CategoryPatents: {}, CategoryFellowships: {},
	CategoryInternationalAgreements: {}, CategoryLocalAgreements: {},
}

// workshopTypes are the sub-types kept inside the workshops category.
var workshopTypes = map[string]struct{}{"seminar": {}, "workshop": {}, "panel": {}, "lecture": {}}

// CategoryForType maps a legacy activity type onto its FS 003 category, so callers
// that still send only activity_type keep working.
func CategoryForType(t string) string {
	if _, ok := workshopTypes[t]; ok {
		return CategoryWorkshops
	}
	switch t {
	case "course":
		return CategoryCourses
	case "conference":
		return CategoryConferences
	case "other":
		return CategoryCommunityService
	}
	return ""
}

// TypeForCategory keeps activity_type consistent with the category: the workshops
// category keeps a valid sub-type (seminar, panel…), every other category has one type.
func TypeForCategory(category, requested string) string {
	switch category {
	case CategoryWorkshops:
		if _, ok := workshopTypes[requested]; ok {
			return requested
		}
		return "workshop"
	case CategoryCourses:
		return "course"
	case CategoryConferences:
		return "conference"
	default:
		return "other"
	}
}

// CategoryForIndexDB places a verified journal publication on the form: local
// journals on sheet 4, Scopus / Web of Science on sheet 5, other international on sheet 6.
func CategoryForIndexDB(indexDB string) string {
	switch indexDB {
	case "local":
		return CategoryLocalResearch
	case "scopus", "wos":
		return CategoryIndexedResearch
	case "international_unindexed":
		return CategoryInternationalResearch
	}
	return ""
}
