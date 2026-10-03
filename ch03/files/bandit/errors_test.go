package bandit

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestArmErrorMessageNamesTheArmAndTheCause(t *testing.T) {
	err := &ArmError{Arm: 3, Err: errors.New("boom")}
	if got := err.Error(); got != "arm 3: boom" {
		t.Errorf("Error() = %q, want %q", got, "arm 3: boom")
	}
}

func TestArmErrorUnwrapsToItsCause(t *testing.T) {
	err := &ArmError{Arm: 1, Err: ErrInvalidProbability}
	if !errors.Is(err, ErrInvalidProbability) {
		t.Error("errors.Is cannot see through ArmError: Unwrap is missing or wrong")
	}
	if errors.Is(err, ErrNoArms) {
		t.Error("errors.Is matched an unrelated sentinel")
	}
}

func TestArmErrorSurvivesWrapping(t *testing.T) {
	inner := &ArmError{Arm: 2, Err: ErrInvalidProbability}
	wrapped := fmt.Errorf("outer context: %w", inner)

	var ae *ArmError
	if !errors.As(wrapped, &ae) || ae.Arm != 2 {
		t.Fatalf("errors.As did not find the *ArmError inside %v", wrapped)
	}
	if !errors.Is(wrapped, ErrInvalidProbability) {
		t.Error("errors.Is lost the sentinel behind the wrapper")
	}
	if !strings.Contains(wrapped.Error(), "arm 2") {
		t.Errorf("message %q should include the arm", wrapped)
	}
}
