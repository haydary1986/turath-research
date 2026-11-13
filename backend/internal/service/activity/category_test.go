package activity

import (
	"slices"
	"testing"
)

func TestCategoryRegistry(t *testing.T) {
	if len(Categories) != 13 {
		t.Fatalf("FS 003 has 13 sheets, got %d categories", len(Categories))
	}
	for _, c := range Categories {
		if !IsCategory(c) {
			t.Errorf("%s not registered", c)
		}
	}
	if IsCategory("sheet8") || IsCategory("") {
		t.Error("unknown categories must be rejected")
	}
	var inst []string
	for _, c := range Categories {
		if IsInstitutional(c) {
			inst = append(inst, c)
		}
	}
	want := []string{CategoryInfrastructure, CategoryInternationalAgreements, CategoryLocalAgreements}
	if !slices.Equal(inst, want) {
		t.Errorf("institutional = %v, want %v", inst, want)
	}
}

func TestCategoryForType(t *testing.T) {
	cases := map[string]string{
		"seminar": CategoryWorkshops, "workshop": CategoryWorkshops, "panel": CategoryWorkshops,
		"lecture": CategoryWorkshops, "course": CategoryCourses, "conference": CategoryConferences,
		"other": CategoryCommunityService, "party": "",
	}
	for typ, want := range cases {
		if got := CategoryForType(typ); got != want {
			t.Errorf("%s → %q, want %q", typ, got, want)
		}
	}
}

func TestTypeForCategory(t *testing.T) {
	cases := []struct{ category, requested, want string }{
		{CategoryWorkshops, "panel", "panel"},
		{CategoryWorkshops, "course", "workshop"},
		{CategoryWorkshops, "", "workshop"},
		{CategoryCourses, "seminar", "course"},
		{CategoryConferences, "", "conference"},
		{CategoryPatents, "seminar", "other"},
		{CategoryInfrastructure, "", "other"},
	}
	for _, c := range cases {
		if got := TypeForCategory(c.category, c.requested); got != c.want {
			t.Errorf("%s/%s → %q, want %q", c.category, c.requested, got, c.want)
		}
	}
}

func TestCategoryForIndexDB(t *testing.T) {
	cases := map[string]string{
		"local": CategoryLocalResearch, "scopus": CategoryIndexedResearch, "wos": CategoryIndexedResearch,
		"international_unindexed": CategoryInternationalResearch, "": "", "other": "",
	}
	for db, want := range cases {
		if got := CategoryForIndexDB(db); got != want {
			t.Errorf("%q → %q, want %q", db, got, want)
		}
	}
}

func TestNormalizeLegacyTypeOnly(t *testing.T) {
	got := Normalize(Input{Type: "seminar", Location: " القاعة "})
	if got.Category != CategoryWorkshops || got.Type != "seminar" {
		t.Errorf("legacy seminar: %+v", got)
	}
	if got.CategoryDetail != "القاعة" || got.Location != "القاعة" {
		t.Errorf("venue mirrors into the form column: %+v", got)
	}
}

func TestNormalizeCategoryWins(t *testing.T) {
	count := 3
	got := Normalize(Input{Category: CategoryBooks, Type: "seminar", CategoryDetail: "دار الكتب", ItemCount: &count})
	if got.Type != "other" {
		t.Errorf("type follows the category: %q", got.Type)
	}
	if got.ItemCount != nil {
		t.Error("only infrastructure keeps a count")
	}
	if got.Location != "" {
		t.Error("a publisher is not a venue")
	}
	in := Input{Category: CategoryCourses, CategoryDetail: "مركز التعليم المستمر"}
	if out := Normalize(in); out.Location != "مركز التعليم المستمر" || in.Location != "" {
		t.Errorf("course venue fills location without touching the input: %+v", out)
	}
}

func TestValidateCategoryRules(t *testing.T) {
	base := Input{TitleAr: "عنوان", ActivityDate: "2026-11-02"}
	count := func(n int) *int { return &n }
	cases := []struct {
		name    string
		in      Input
		college bool
		want    string
	}{
		{"no category nor type", base, true, "category:required"},
		{"unknown category", with(base, func(i *Input) { i.Category = "sheet8" }), true, "category:invalid"},
		{"infrastructure needs count", with(base, func(i *Input) { i.Category = CategoryInfrastructure }), false, "item_count:required"},
		{"infrastructure count positive", with(base, func(i *Input) { i.Category = CategoryInfrastructure; i.ItemCount = count(0) }), false, "item_count:invalid"},
		{"book needs publisher", with(base, func(i *Input) { i.Category = CategoryBooks }), true, "category_detail:required"},
		{"agreement needs party", with(base, func(i *Input) { i.Category = CategoryLocalAgreements }), false, "category_detail:required"},
		{"personal needs college", with(base, func(i *Input) { i.Category = CategoryPatents; i.CategoryDetail = "x" }), false, "college:required"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			errs := Validate(Normalize(c.in), c.college)
			if !slices.Contains(errs, c.want) {
				t.Fatalf("want %q in %v", c.want, errs)
			}
		})
	}

	ok := []Input{
		with(base, func(i *Input) { i.Category = CategoryInfrastructure; i.ItemCount = count(4) }),
		with(base, func(i *Input) { i.Category = CategoryInternationalAgreements; i.CategoryDetail = "جامعة لندن" }),
		with(base, func(i *Input) { i.Type = "seminar" }),
	}
	for _, in := range ok {
		college := in.Category == "" // legacy seminar is personal and needs one
		if errs := Validate(Normalize(in), college); len(errs) != 0 {
			t.Errorf("%+v: unexpected %v", in, errs)
		}
	}
}

func with(in Input, mutate func(*Input)) Input {
	mutate(&in)
	return in
}
