package handler

import (
	"reflect"
	"testing"
)

func TestValidateNewUser(t *testing.T) {
	cases := []struct {
		name string
		in   createUserInput
		want []string
	}{
		{"valid, normalized", createUserInput{Email: " Ali.Hassan@UOTURATH.edu.iq ", NameAr: " علي ", Role: "college_dean"}, nil},
		{"missing fields", createUserInput{}, []string{"email:required", "name_ar:required", "role:invalid"}},
		{"other domain", createUserInput{Email: "a@gmail.com", NameAr: "ع", Role: "researcher"}, []string{"email:domain_not_allowed"}},
		{"look-alike domain", createUserInput{Email: "a@evil-uoturath.edu.iq", NameAr: "ع", Role: "researcher"}, []string{"email:domain_not_allowed"}},
		{"display-name form rejected", createUserInput{Email: "Ali <a@uoturath.edu.iq>", NameAr: "ع", Role: "researcher"}, []string{"email:invalid"}},
		{"legacy role", createUserInput{Email: "a@uoturath.edu.iq", NameAr: "ع", Role: "viewer"}, []string{"role:invalid"}},
	}
	for _, tc := range cases {
		out, got := validateNewUser(tc.in, "uoturath.edu.iq")
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: problems %v, want %v", tc.name, got, tc.want)
		}
		if tc.want == nil && (out.Email != "ali.hassan@uoturath.edu.iq" || out.NameAr != "علي") {
			t.Errorf("%s: not normalized: %+v", tc.name, out)
		}
	}
}
