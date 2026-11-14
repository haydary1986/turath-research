package publicsite

import (
	"slices"
	"strings"
	"testing"
)

func TestDepartmentBlock(t *testing.T) {
	base := NormalizeContent(Content{HeroTitle: "x"})
	if base.Department.Goals == nil || base.Department.Tasks == nil {
		t.Fatal("empty lists must serialise as [] not null")
	}

	c := base
	c.Department = Department{Vision: "  رؤية  ", Goals: []DeptItem{{Text: " هدف "}}, Tasks: []DeptItem{{Text: "مهمة"}}}
	n := NormalizeContent(c)
	if n.Department.Vision != "رؤية" || n.Department.Goals[0].Text != "هدف" {
		t.Fatalf("not cleaned: %+v", n.Department)
	}
	if problems := ValidateContent(n); len(problems) != 0 {
		t.Fatalf("valid block rejected: %v", problems)
	}

	bad := base
	bad.Department = Department{
		Mission: strings.Repeat("م", MaxDeptStatement+1),
		Goals:   []DeptItem{{Text: ""}},
		Tasks:   slices.Repeat([]DeptItem{{Text: "t"}}, MaxDeptItems+1),
	}
	problems := ValidateContent(NormalizeContent(bad))
	for _, want := range []string{"dept_mission:too_long", "dept_goal:required", "dept_tasks:too_many"} {
		if !slices.Contains(problems, want) {
			t.Errorf("missing %s in %v", want, problems)
		}
	}
}
