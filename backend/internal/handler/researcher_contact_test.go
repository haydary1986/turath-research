package handler

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestContactUpdateNormalized(t *testing.T) {
	phone, empty, bad := " 0770 123 4567 ", "", "not-an-email"
	got, problems := contactUpdate{Phone: &phone, PhoneSecondary: &empty}.normalized()
	if len(problems) != 0 || *got.Phone != "07701234567" || *got.PhoneSecondary != "" || got.RecoveryEmail != nil {
		t.Errorf("got %+v, problems %v", got, problems)
	}
	short := "0770"
	_, problems = contactUpdate{Phone: &short, PhoneSecondary: &bad, RecoveryEmail: &bad}.normalized()
	want := "phone:invalid_phone,phone_secondary:invalid_phone,recovery_email:invalid_email"
	if strings.Join(problems, ",") != want {
		t.Errorf("problems = %v", problems)
	}
}

// Contact columns may appear only where access is decided. A new query that selects
// them, a notification that mails recovery_email or an export must fail here first.
func TestContactColumnsStayInGuardedFiles(t *testing.T) {
	allowed := map[string]bool{
		"researcher_contact.go": true, // owner and staff only
		"research_profile.go":   true, // the caller's own record
		"admin_import_row.go":   true, // writes only, admin only
	}
	column := regexp.MustCompile(`\b(phone|phone_secondary|recovery_email)\b`)
	for _, dir := range []string{".", "../service/sync", "../service/portal", "../service/mailer", "../service/ai"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			base := filepath.Base(file)
			if strings.HasSuffix(base, "_test.go") || (dir == "." && allowed[base]) {
				continue
			}
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if match := column.Find(data); match != nil {
				t.Errorf("%s mentions %s", file, match)
			}
		}
	}
}
