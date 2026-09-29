package ex2

import (
	"errors"
	"fmt"
	"testing"
)

func TestCheckArmOK(t *testing.T) {
	for _, arm := range []int{0, 1, 2} {
		if err := CheckArm(arm, 3); err != nil {
			t.Errorf("CheckArm(%d, 3) = %v, want nil", arm, err)
		}
	}
}

func TestCheckArmErrors(t *testing.T) {
	tests := []struct {
		arm, n  int
		wantMsg string
	}{
		{7, 3, "arm 7 out of range [0, 3)"},
		{-1, 3, "arm -1 out of range [0, 3)"},
		{3, 3, "arm 3 out of range [0, 3)"},
		{0, 0, "arm 0 out of range [0, 0)"},
	}
	for _, tc := range tests {
		err := CheckArm(tc.arm, tc.n)
		if err == nil {
			t.Fatalf("CheckArm(%d, %d) = nil, want an error", tc.arm, tc.n)
		}
		if err.Error() != tc.wantMsg {
			t.Errorf("message = %q, want %q", err.Error(), tc.wantMsg)
		}
		if !errors.Is(err, ErrOutOfRange) {
			t.Errorf("errors.Is(err, ErrOutOfRange) = false")
		}
		var re *RangeError
		if !errors.As(err, &re) || re.Arm != tc.arm || re.N != tc.n {
			t.Errorf("errors.As gave %+v, want Arm=%d N=%d", re, tc.arm, tc.n)
		}
	}
}

func TestCheckArmSurvivesWrapping(t *testing.T) {
	err := fmt.Errorf("step 12: %w", CheckArm(9, 2))
	if !errors.Is(err, ErrOutOfRange) {
		t.Error("errors.Is should see through fmt.Errorf's %w")
	}
	var re *RangeError
	if !errors.As(err, &re) || re.Arm != 9 {
		t.Error("errors.As should see through fmt.Errorf's %w")
	}
}

// A nil *RangeError stored in an error would be a non-nil interface: make
// sure CheckArm returns a true nil on success.
func TestCheckArmNilIsNil(t *testing.T) {
	var err error = CheckArm(0, 1)
	if err != nil {
		t.Fatalf("got %#v, want a nil error", err)
	}
}
