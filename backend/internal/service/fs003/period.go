package fs003

import (
	"fmt"
	"time"
	// The runtime image ships no zoneinfo; the form's days are Baghdad days.
	_ "time/tzdata"
)

// Iraqi month names as the ministry writes them.
var monthsAr = [12]string{
	"كانون الثاني", "شباط", "آذار", "نيسان", "أيار", "حزيران",
	"تموز", "آب", "أيلول", "تشرين الأول", "تشرين الثاني", "كانون الأول",
}

// Baghdad is the time zone of every date on the form.
var Baghdad = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Baghdad")
	if err != nil {
		return time.FixedZone("Asia/Baghdad", 3*60*60) // Iraq has kept UTC+3 without DST since 2008
	}
	return loc
}()

const isoDate = "2006-01-02"

// FormatDate turns a YYYY-MM-DD day into the form's day/month/year; anything else is returned as is.
func FormatDate(iso string) string {
	d, err := time.Parse(isoDate, iso)
	if err != nil {
		return iso
	}
	return d.Format("02/01/2006")
}

// PeriodPhrase completes «… لشهر ___ عام ___» for the reporting period: one calendar
// month reads «لشهر <month> عام <year>», a calendar year «لعام <year>», any other
// range «للمدة من <from> إلى <to>».
func PeriodPhrase(from, to time.Time) string {
	from = dateOnly(from)
	to = dateOnly(to)
	if from.Day() == 1 && sameMonth(from, to) && to.AddDate(0, 0, 1).Month() != to.Month() {
		return fmt.Sprintf("لشهر %s عام %d", monthsAr[from.Month()-1], from.Year())
	}
	if from.Year() == to.Year() && from.YearDay() == 1 && to.Month() == time.December && to.Day() == 31 {
		return fmt.Sprintf("لعام %d", from.Year())
	}
	return fmt.Sprintf("للمدة من %s إلى %s", from.Format("02/01/2006"), to.Format("02/01/2006"))
}

func sameMonth(a, b time.Time) bool { return a.Year() == b.Year() && a.Month() == b.Month() }

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
