package service

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/taskmgr/internal/models"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/websocket"
	"github.com/google/uuid"
)

const MaxAttachmentSize = 10 << 20 // 10 MB

var (
	ErrFileTooLarge       = errors.New("file exceeds maximum allowed size of 10MB")
	ErrUnsupportedFile    = errors.New("unsupported file type")
)

var allowedExtensions = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
	".pdf": true, ".txt": true, ".md": true, ".csv": true, ".json": true,
	".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".zip": true,
}

type AttachmentService struct {
	attRepo     *repository.AttachmentRepository
	taskRepo    *repository.TaskRepository
	projectRepo *repository.ProjectRepository
	hub         *websocket.Hub
	uploadDir   string
}

func NewAttachmentService(
	attRepo *repository.AttachmentRepository,
	taskRepo *repository.TaskRepository,
	projectRepo *repository.ProjectRepository,
	hub *websocket.Hub,
	uploadDir string,
) *AttachmentService {
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	_ = os.MkdirAll(uploadDir, 0750)
	return &AttachmentService{
		attRepo:     attRepo,
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		hub:         hub,
		uploadDir:   uploadDir,
	}
}

func (s *AttachmentService) Upload(
	ctx context.Context,
	userID, projectID, taskID uuid.UUID,
	originalName, contentType string,
	size int64,
	reader io.Reader,
) (*models.TaskAttachment, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	if _, err := s.taskRepo.GetByID(ctx, projectID, taskID); err != nil {
		return nil, err
	}
	if size > MaxAttachmentSize {
		return nil, ErrFileTooLarge
	}

	safeOriginal := filepath.Base(strings.ReplaceAll(originalName, "\\", "/"))
	ext := strings.ToLower(filepath.Ext(safeOriginal))
	if !allowedExtensions[ext] {
		return nil, ErrUnsupportedFile
	}

	storageKey := uuid.New().String() + ext
	destPath := filepath.Join(s.uploadDir, storageKey)

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
	if err != nil {
		return nil, err
	}
	written, err := io.Copy(out, io.LimitReader(reader, MaxAttachmentSize+1))
	out.Close()
	if err != nil {
		_ = os.Remove(destPath)
		return nil, err
	}
	if written > MaxAttachmentSize {
		_ = os.Remove(destPath)
		return nil, ErrFileTooLarge
	}

	att, err := s.attRepo.Create(ctx, &models.TaskAttachment{
		ProjectID:    projectID,
		TaskID:       taskID,
		UploadedBy:   &userID,
		OriginalName: safeOriginal,
		StorageKey:   storageKey,
		ContentType:  contentType,
		SizeBytes:    written,
	})
	if err != nil {
		_ = os.Remove(destPath)
		return nil, err
	}

	if s.hub != nil {
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventAttachmentChanged,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload:   att,
		})
	}
	return att, nil
}

func (s *AttachmentService) List(ctx context.Context, userID, projectID, taskID uuid.UUID) ([]*models.TaskAttachment, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, err
	}
	return s.attRepo.ListByTask(ctx, projectID, taskID)
}

func (s *AttachmentService) GetFileForDownload(ctx context.Context, userID, projectID, taskID, attachmentID uuid.UUID) (*models.TaskAttachment, string, error) {
	if _, err := s.projectRepo.GetMemberRole(ctx, projectID, userID); err != nil {
		return nil, "", err
	}
	att, err := s.attRepo.GetByID(ctx, projectID, taskID, attachmentID)
	if err != nil {
		return nil, "", err
	}
	fullPath := filepath.Join(s.uploadDir, filepath.Base(att.StorageKey))
	return att, fullPath, nil
}

func (s *AttachmentService) Delete(ctx context.Context, userID, projectID, taskID, attachmentID uuid.UUID) error {
	role, err := s.projectRepo.GetMemberRole(ctx, projectID, userID)
	if err != nil {
		return err
	}
	att, err := s.attRepo.GetByID(ctx, projectID, taskID, attachmentID)
	if err != nil {
		return err
	}
	if role != models.RoleAdmin && (att.UploadedBy == nil || *att.UploadedBy != userID) {
		return ErrForbidden
	}
	if err := s.attRepo.Delete(ctx, projectID, taskID, attachmentID); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(s.uploadDir, filepath.Base(att.StorageKey)))

	if s.hub != nil {
		s.hub.Broadcast(&models.RealtimeEvent{
			Type:      "event",
			Event:     models.EventAttachmentChanged,
			EventID:   uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Payload:   map[string]interface{}{"task_id": taskID, "attachment_id": attachmentID},
		})
	}
	return nil
}
