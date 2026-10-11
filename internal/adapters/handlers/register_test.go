package handlers

import (
	"testing"
	"time"
)

func TestSignupLimiter(t *testing.T) {
	l := newSignupLimiter(5, time.Minute)
	if !l.allow("a", 3) || !l.allow("a", 2) {
		t.Fatal("first 5 people should pass")
	}
	if l.allow("a", 1) {
		t.Fatal("6th person should be blocked")
	}
	if !l.allow("b", 5) {
		t.Fatal("other IPs have their own quota")
	}
	l.refund("a", 2)
	if !l.allow("a", 2) {
		t.Fatal("refunded slots should be usable again")
	}
}
