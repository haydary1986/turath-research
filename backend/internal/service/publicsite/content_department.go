package publicsite

// The department profile block: vision, mission, goals and tasks, as the department
// describes itself. Every text is bilingual; English may be left empty.

const (
	MaxDeptStatement = 1500
	MaxDeptItems     = 20
	MaxDeptItem      = 400
)

// DeptItem is one entry of the goals or tasks list.
type DeptItem struct {
	Text   string `json:"text"`
	TextEn string `json:"text_en"`
}

// Department is the department's vision, mission, goals and tasks.
type Department struct {
	Vision    string     `json:"vision"`
	VisionEn  string     `json:"vision_en"`
	Mission   string     `json:"mission"`
	MissionEn string     `json:"mission_en"`
	Goals     []DeptItem `json:"goals"`
	Tasks     []DeptItem `json:"tasks"`
}

func normalizeDeptItems(in []DeptItem) []DeptItem {
	out := make([]DeptItem, 0, len(in))
	for _, item := range in {
		out = append(out, DeptItem{Text: cleanText(item.Text), TextEn: cleanText(item.TextEn)})
	}
	return out
}

func normalizeDepartment(d Department) Department {
	return Department{
		Vision: cleanText(d.Vision), VisionEn: cleanText(d.VisionEn),
		Mission: cleanText(d.Mission), MissionEn: cleanText(d.MissionEn),
		Goals: normalizeDeptItems(d.Goals), Tasks: normalizeDeptItems(d.Tasks),
	}
}

func validateDepartment(p *problems, d Department) {
	p.text("dept_vision", d.Vision, false, MaxDeptStatement)
	p.text("dept_vision_en", d.VisionEn, false, MaxDeptStatement)
	p.text("dept_mission", d.Mission, false, MaxDeptStatement)
	p.text("dept_mission_en", d.MissionEn, false, MaxDeptStatement)
	for list, items := range map[string][]DeptItem{"dept_goal": d.Goals, "dept_task": d.Tasks} {
		if len(items) > MaxDeptItems {
			p.add(list+"s", CodeTooMany)
		}
		for _, item := range items {
			p.text(list, item.Text, true, MaxDeptItem)
			p.text(list+"_en", item.TextEn, false, MaxDeptItem)
		}
	}
}
