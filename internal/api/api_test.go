package api

import "testing"

func TestValidateDate(t *testing.T) {
	valid := []string{"2026-01-01", "2026-12-31", "2020-02-29"}
	for _, s := range valid {
		if err := ValidateDate(s); err != nil {
			t.Errorf("ValidateDate(%q) = %v, want nil", s, err)
		}
	}
	invalid := []string{"2026-13-01", "2026-01-40", "01-01-2026", "2026/01/01", "", "today"}
	for _, s := range invalid {
		if err := ValidateDate(s); err == nil {
			t.Errorf("ValidateDate(%q) = nil, want error", s)
		}
	}
}

func TestValidateRange(t *testing.T) {
	if err := ValidateRange("2026-01-01", "2026-01-02"); err != nil {
		t.Errorf("in-order range rejected: %v", err)
	}
	if err := ValidateRange("2026-02-01", "2026-01-01"); err == nil {
		t.Error("reversed range accepted, want error")
	}
	if err := ValidateRange("", "2026-01-01"); err != nil {
		t.Errorf("open-ended range rejected: %v", err)
	}
}

func TestExitCode(t *testing.T) {
	if got := ExitCode(nil); got != ExitOK {
		t.Errorf("ExitCode(nil) = %d, want %d", got, ExitOK)
	}
	if got := ExitCode(coded(ExitRateLimit, "boom")); got != ExitRateLimit {
		t.Errorf("ExitCode(coded 5) = %d, want %d", got, ExitRateLimit)
	}
}

func TestRequireID(t *testing.T) {
	if err := requireID("accountId", ""); err == nil {
		t.Error("empty id accepted")
	}
	if err := requireID("accountId", "a/b"); err == nil {
		t.Error("id with slash accepted")
	}
	if err := requireID("accountId", "3353431574710163189587446"); err != nil {
		t.Errorf("valid id rejected: %v", err)
	}
}
