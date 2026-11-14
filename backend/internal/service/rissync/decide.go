// Package rissync keeps the researcher profiles of this portal and of the RIS
// (ris.uoturath.edu.iq) in step. Each synced field remembers the value both sides
// last agreed on; only a side that moved away from it since is copied across, so
// edits never bounce back and forth.
package rissync

// Action is what one field needs after comparing both sides with the last agreed value.
type Action int

const (
	// Nothing: both sides still hold the agreed value.
	Nothing Action = iota
	// ToRIS copies the portal value into RIS.
	ToRIS
	// ToPortal copies the RIS value into the portal.
	ToPortal
	// Agree records a value both sides reached independently as the new agreed value.
	Agree
	// Review leaves the field as it is and asks an admin to choose.
	Review
)

func (a Action) String() string {
	return [...]string{"nothing", "to_ris", "to_portal", "agree", "review"}[a]
}

// Policy is how a field is reconciled the first time a researcher is synced.
type Policy int

const (
	// PortalOfficial: the portal (staff sheet) value overwrites RIS on the first sync.
	PortalOfficial Policy = iota
	// FillGaps: an empty side takes the other's value; two different values go to review.
	FillGaps
)

// Values are one field's canonical values on each side; "" means empty.
type Values struct {
	Portal, RIS string
	// Last is the last agreed value; HasLast is false before the first sync.
	Last    string
	HasLast bool
	// PortalNewer reports that the portal record changed after the RIS one.
	PortalNewer bool
}

// Decide chooses the action for one field.
func Decide(v Values, policy Policy) Action {
	if !v.HasLast {
		return decideFirst(v, policy)
	}
	portalMoved, risMoved := v.Portal != v.Last, v.RIS != v.Last
	switch {
	case !portalMoved && !risMoved:
		return Nothing
	case portalMoved && !risMoved:
		return ToRIS
	case risMoved && !portalMoved:
		return ToPortal
	case v.Portal == v.RIS:
		return Agree
	case v.PortalNewer:
		// Both sides were edited since the last sync: the most recent edit wins.
		return ToRIS
	default:
		return ToPortal
	}
}

func decideFirst(v Values, policy Policy) Action {
	switch {
	case v.Portal == v.RIS:
		return Agree
	case policy == PortalOfficial:
		if v.Portal == "" {
			return ToPortal // nothing official to impose; keep what RIS has
		}
		return ToRIS
	case v.RIS == "":
		return ToRIS
	case v.Portal == "":
		return ToPortal
	default:
		return Review
	}
}
