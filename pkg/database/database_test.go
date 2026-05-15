package database

import (
	"testing"
	"unicode"
)

func TestRandomPassword(t *testing.T) {
	const n = 64
	pwd := randomPassword(n)

	if len(pwd) != n {
		t.Errorf("expected length %d, got %d", n, len(pwd))
	}

	var hasUpper, hasLower, hasDigit bool
	for _, ch := range pwd {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}

	if !hasUpper {
		t.Error("password missing at least one uppercase letter")
	}
	if !hasLower {
		t.Error("password missing at least one lowercase letter")
	}
	if !hasDigit {
		t.Error("password missing at least one digit")
	}
}
