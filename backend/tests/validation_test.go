package tests

import (
	"testing"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/validation"
)

func TestValidateRegisterInput(t *testing.T) {
	// Valid case
	err := validation.ValidateRegisterInput("Alice Smith", "alice@example.com", "Password123!")
	if err != nil {
		t.Errorf("expected valid registration, got %v", err)
	}

	// Empty name
	err = validation.ValidateRegisterInput("", "alice@example.com", "Password123!")
	if err != validation.ErrNameRequired {
		t.Errorf("expected ErrNameRequired, got %v", err)
	}

	// Invalid email
	err = validation.ValidateRegisterInput("Alice", "invalid-email", "Password123!")
	if err != validation.ErrInvalidEmail {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}

	// Short password
	err = validation.ValidateRegisterInput("Alice", "alice@example.com", "short")
	if err != validation.ErrPasswordTooShort {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestValidateTaskInput(t *testing.T) {
	// Valid
	err := validation.ValidateTaskInput("Setup database", models.StatusTodo, models.PriorityHigh, nil)
	if err != nil {
		t.Errorf("expected valid task, got %v", err)
	}

	// Empty title
	err = validation.ValidateTaskInput("", models.StatusTodo, models.PriorityHigh, nil)
	if err != validation.ErrTitleRequired {
		t.Errorf("expected ErrTitleRequired, got %v", err)
	}

	// Invalid status
	err = validation.ValidateTaskInput("Fix bug", "UNKNOWN_STATUS", models.PriorityLow, nil)
	if err != validation.ErrInvalidStatus {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}

	// Invalid priority
	err = validation.ValidateTaskInput("Fix bug", models.StatusTodo, "SUPER_URGENT", nil)
	if err != validation.ErrInvalidPriority {
		t.Errorf("expected ErrInvalidPriority, got %v", err)
	}
}

func TestValidateRole(t *testing.T) {
	if err := validation.ValidateRole(models.RoleAdmin); err != nil {
		t.Errorf("expected admin role to be valid, got %v", err)
	}
	if err := validation.ValidateRole(models.RoleMember); err != nil {
		t.Errorf("expected member role to be valid, got %v", err)
	}
	if err := validation.ValidateRole("guest"); err != validation.ErrInvalidRole {
		t.Errorf("expected ErrInvalidRole for guest, got %v", err)
	}
}
