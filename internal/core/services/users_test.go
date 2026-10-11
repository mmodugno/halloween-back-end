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

func TestJoinNames(t *testing.T) {
	cases := map[string][]string{
		"":               nil,
		"Lu":             {"Lu"},
		"Lu y Tomi":      {"Lu", "", "Tomi"},
		"Lu, Sol y Tomi": {"Lu", "Sol", "Tomi"},
	}
	for want, names := range cases {
		if got := JoinNames(names); got != want {
			t.Errorf("JoinNames(%q) = %q, want %q", names, got, want)
		}
	}
}

func TestCleanNames(t *testing.T) {
	got, err := cleanNames("  Lu  Bruja ", []string{"", " Tomi ", "  "})
	if err != nil || len(got) != 2 || got[0] != "Lu Bruja" || got[1] != "Tomi" {
		t.Fatalf("cleanNames = %q, %v", got, err)
	}
	if _, err := cleanNames(" ", nil); err != ErrMissingName {
		t.Errorf("empty name: got %v", err)
	}
	if _, err := cleanNames("Lu", []string{"lu"}); err != ErrDuplicateName {
		t.Errorf("repeated name: got %v", err)
	}
	if _, err := cleanNames("Lu", make9("x")); err != ErrTooManyPartners {
		t.Errorf("too many: got %v", err)
	}
}

func make9(prefix string) []string {
	out := make([]string, 0, MaxPartners+1)
	for i := 0; i <= MaxPartners; i++ {
		out = append(out, prefix+string(rune('a'+i)))
	}
	return out
}
