package validation

import (
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/google/uuid"
)

var (
	ErrInvalidEmail    = errors.New("invalid email address format")
	ErrPasswordTooShort = errors.New("password must be at least 8 characters long")
	ErrNameRequired    = errors.New("name is required and cannot be empty")
	ErrTitleRequired   = errors.New("title is required and cannot be empty")
	ErrInvalidStatus   = errors.New("invalid task status: must be TODO, IN_PROGRESS, or DONE")
	ErrInvalidPriority = errors.New("invalid task priority: must be LOW, MEDIUM, or HIGH")
	ErrInvalidRole     = errors.New("invalid role: must be admin or member")
	ErrContentRequired = errors.New("comment content cannot be empty")
)

func ValidateRegisterInput(name, email, password string) error {
	if strings.TrimSpace(name) == "" {
		return ErrNameRequired
	}
	if len(strings.TrimSpace(name)) > 100 {
		return errors.New("name cannot exceed 100 characters")
	}
	if err := ValidateEmail(email); err != nil {
		return err
	}
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	return nil
}

func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return ErrInvalidEmail
	}
	return nil
}

func ValidateProjectInput(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrTitleRequired
	}
	if len(strings.TrimSpace(name)) > 150 {
		return errors.New("project name cannot exceed 150 characters")
	}
	return nil
}

func ValidateTaskInput(title, status, priority string, dueDate *time.Time) error {
	if strings.TrimSpace(title) == "" {
		return ErrTitleRequired
	}
	if len(strings.TrimSpace(title)) > 255 {
		return errors.New("task title cannot exceed 255 characters")
	}
	if status != "" && status != models.StatusTodo && status != models.StatusInProgress && status != models.StatusDone {
		return ErrInvalidStatus
	}
	if priority != "" && priority != models.PriorityLow && priority != models.PriorityMedium && priority != models.PriorityHigh {
		return ErrInvalidPriority
	}
	return nil
}

// ValidateTitleOnly validates only the task title field.
func ValidateTitleOnly(title string) error {
	if strings.TrimSpace(title) == "" {
		return ErrTitleRequired
	}
	if len(strings.TrimSpace(title)) > 255 {
		return errors.New("task title cannot exceed 255 characters")
	}
	return nil
}

// ValidateStatusOnly validates only the task status field.
func ValidateStatusOnly(status string) error {
	if status != models.StatusTodo && status != models.StatusInProgress && status != models.StatusDone {
		return ErrInvalidStatus
	}
	return nil
}

// ValidatePriorityOnly validates only the task priority field.
func ValidatePriorityOnly(priority string) error {
	if priority != models.PriorityLow && priority != models.PriorityMedium && priority != models.PriorityHigh {
		return ErrInvalidPriority
	}
	return nil
}

func ValidateRole(role string) error {
	if role != models.RoleAdmin && role != models.RoleMember {
		return ErrInvalidRole
	}
	return nil
}

func ValidateCommentInput(content string) error {
	if strings.TrimSpace(content) == "" {
		return ErrContentRequired
	}
	if len(content) > 5000 {
		return errors.New("comment content cannot exceed 5000 characters")
	}
	return nil
}

func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(s))
}
