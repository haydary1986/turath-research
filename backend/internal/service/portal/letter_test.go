package portal

import (
	"reflect"
	"testing"
)

func TestRenderBody(t *testing.T) {
	got := RenderBody("تقديرًا لجهودكم في بحث {{research_title}} المنشور في {{journal_name}}.", map[string]string{
		"research_title": "AI-based Optimization",
		"journal_name":   "Energy and Buildings",
	})
	want := "تقديرًا لجهودكم في بحث AI-based Optimization المنشور في Energy and Buildings."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRenderBodyKeepsUnknownPlaceholders(t *testing.T) {
	got := RenderBody("{{a}} and {{b}}", map[string]string{"a": "1"})
	if got != "1 and {{b}}" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderBodyDoesNotReinterpretValues(t *testing.T) {
	got := RenderBody("{{a}} {{b}}", map[string]string{"a": "{{b}}", "b": "X"})
	if got != "{{b}} X" {
		t.Fatalf("values must not be expanded recursively, got %q", got)
	}
}

func TestUnknownPlaceholders(t *testing.T) {
	got := UnknownPlaceholders("{{researcher_name}} {{typo}} {{research_title}} {{typo}}")
	want := []string{"typo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if len(UnknownPlaceholders("no placeholders")) != 0 {
		t.Fatal("expected none")
	}
}

func TestFormatLetterNo(t *testing.T) {
	if got := FormatLetterNo(1245, 2026); got != "1245/2026" {
		t.Fatalf("got %s", got)
	}
}
