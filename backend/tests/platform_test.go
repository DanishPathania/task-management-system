package tests

import (
	"testing"

	"github.com/example/taskmgr/internal/service"
)

func TestAttachmentConstantsAndErrors(t *testing.T) {
	if service.MaxAttachmentSize != 10<<20 {
		t.Fatalf("expected MaxAttachmentSize 10MB, got %d", service.MaxAttachmentSize)
	}
	if service.ErrFileTooLarge == nil || service.ErrUnsupportedFile == nil {
		t.Fatalf("expected attachment error sentinels to be defined")
	}
}

func TestNotificationTypeConstants(t *testing.T) {
	types := []string{
		service.NotifTaskAssigned,
		service.NotifTaskDue,
		service.NotifTaskOverdue,
		service.NotifMention,
		service.NotifComment,
		service.NotifMemberAdded,
		service.NotifStatusChanged,
	}
	seen := make(map[string]bool)
	for _, typ := range types {
		if typ == "" {
			t.Fatalf("notification type constant must not be empty")
		}
		if seen[typ] {
			t.Fatalf("duplicate notification type constant: %s", typ)
		}
		seen[typ] = true
	}
}
