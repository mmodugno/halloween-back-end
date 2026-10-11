package services

import "testing"

func TestPassBase(t *testing.T) {
	cases := map[string]string{
		"Maga":       "maga",
		"María José": "mariajose",
		"Ñandú 2":    "nandu2",
		"  !!  ":     "user",
		"Lu-Bruja.":  "lubruja",
	}
	for name, want := range cases {
		if got := passBase(name); got != want {
			t.Errorf("passBase(%q) = %q, want %q", name, got, want)
		}
	}
}
