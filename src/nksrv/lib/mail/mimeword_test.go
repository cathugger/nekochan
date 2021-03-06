package mail

import (
	gmail "net/mail"
	"testing"
)

func TestFormatAddress(t *testing.T) {
	type testcase struct {
		name, email, result string
	}
	tests := []testcase{
		{name: "", email: "a@b", result: "<a@b>"},
		{name: "a", email: "a@b", result: "a <a@b>"},
		{name: "a b c", email: "a@b", result: "a b c <a@b>"},
		{name: "a b\tc", email: "a@b", result: "\"a b\tc\" <a@b>"},
		{name: " a  b  c ", email: "a@b", result: "\" a  b  c \" <a@b>"},
		{name: "a@b", email: "a@b", result: "\"a@b\" <a@b>"},
		{name: "a\"\"\"b", email: "a@b", result: "\"a\\\"\\\"\\\"b\" <a@b>"},
		{name: "a\\\\\\b", email: "a@b", result: "\"a\\\\\\\\\\\\b\" <a@b>"},
	}
	for i := range tests {
		res := FormatAddress(tests[i].name, tests[i].email)
		wantres := tests[i].result
		if res != wantres {
			t.Errorf("FormatAddress returned %q wanted %q", res, wantres)
		}
		a, e := gmail.ParseAddress(res)
		if e != nil {
			t.Errorf("ParseAddressX err: %v", e)
		}
		if a.Name != tests[i].name {
			t.Errorf("ParseAddressX name %q wanted %q", a.Name, tests[i].name)
		}
		if a.Address != tests[i].email {
			t.Errorf("ParseAddressX email %q wanted %q", a.Address, tests[i].email)
		}
	}
}
