package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/example/taskmgr/internal/database"
	"github.com/example/taskmgr/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
)

type NotificationRepository struct {
	db *database.DB
}

func NewNotificationRepository(db *database.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, n *models.Notification, dedupeKey *string) (*models.Notification, bool, error) {
	query := `
		INSERT INTO notifications (user_id, project_id, task_id, actor_id, type, title, message, is_read, dedupe_key, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, FALSE, $8, $9)
		ON CONFLICT (user_id, dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING
		RETURNING id, created_at
	`
	now := time.Now().UTC()
	err := r.db.Pool.QueryRow(
		ctx,
		query,
		n.UserID,
		n.ProjectID,
		n.TaskID,
		n.ActorID,
		n.Type,
		n.Title,
		n.Message,
		dedupeKey,
		now,
	).Scan(&n.ID, &n.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Deduplicated
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("failed to create notification: %w", err)
	}
	n.IsRead = false
	return n, true, nil
}

type PaginatedNotifications struct {
	Notifications []*models.Notification `json:"notifications"`
	UnreadCount   int64                  `json:"unread_count"`
	Pagination    models.Pagination      `json:"pagination"`
}

func (r *NotificationRepository) ListForUser(ctx context.Context, userID uuid.UUID, unreadOnly bool, page, limit int) (*PaginatedNotifications, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var unreadCount int64
	if err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`, userID).Scan(&unreadCount); err != nil {
		return nil, fmt.Errorf("failed to count unread notifications: %w", err)
	}

	countQuery := `SELECT COUNT(*) FROM notifications WHERE user_id = $1`
	listQuery := `
		SELECT id, user_id, project_id, task_id, actor_id, type, title, message, is_read, created_at
		FROM notifications
		WHERE user_id = $1
	`
	if unreadOnly {
		countQuery += ` AND is_read = FALSE`
		listQuery += ` AND is_read = FALSE`
	}

	var totalItems int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, userID).Scan(&totalItems); err != nil {
		return nil, fmt.Errorf("failed to count notifications: %w", err)
	}

	offset := (page - 1) * limit
	listQuery += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	rows, err := r.db.Pool.Query(ctx, listQuery, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	items := make([]*models.Notification, 0)
	for rows.Next() {
		n := &models.Notification{}
		if err := rows.Scan(
			&n.ID,
			&n.UserID,
			&n.ProjectID,
			&n.TaskID,
			&n.ActorID,
			&n.Type,
			&n.Title,
			&n.Message,
			&n.IsRead,
			&n.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		items = append(items, n)
	}

	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &PaginatedNotifications{
		Notifications: items,
		UnreadCount:   unreadCount,
		Pagination: models.Pagination{
			Page:       page,
			Limit:      limit,
			TotalItems: totalItems,
			TotalPages: totalPages,
		},
	}, nil
}

func (r *NotificationRepository) MarkRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	cmdTag, err := r.db.Pool.Exec(
		ctx,
		`UPDATE notifications SET is_read = TRUE WHERE id = $1 AND user_id = $2`,
		notificationID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark notification read: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.Pool.Exec(
		ctx,
		`UPDATE notifications SET is_read = TRUE WHERE user_id = $1 AND is_read = FALSE`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications read: %w", err)
	}
	return nil
}

func (r *NotificationRepository) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.NotificationPreferences, error) {
	prefs := &models.NotificationPreferences{
		UserID:        userID,
		TaskAssigned:  true,
		TaskDue:       true,
		Mentions:      true,
		Comments:      true,
		StatusChanges: true,
		MemberAdded:   true,
		UpdatedAt:     time.Now().UTC(),
	}
	query := `
		SELECT task_assigned, task_due, mentions, comments, status_changes, member_added, updated_at
		FROM notification_preferences
		WHERE user_id = $1
	`
	err := r.db.Pool.QueryRow(ctx, query, userID).Scan(
		&prefs.TaskAssigned,
		&prefs.TaskDue,
		&prefs.Mentions,
		&prefs.Comments,
		&prefs.StatusChanges,
		&prefs.MemberAdded,
		&prefs.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return prefs, nil
		}
		return nil, fmt.Errorf("failed to fetch notification preferences: %w", err)
	}
	return prefs, nil
}

func (r *NotificationRepository) UpdatePreferences(ctx context.Context, prefs *models.NotificationPreferences) (*models.NotificationPreferences, error) {
	now := time.Now().UTC()
	query := `
		INSERT INTO notification_preferences (user_id, task_assigned, task_due, mentions, comments, status_changes, member_added, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			task_assigned = EXCLUDED.task_assigned,
			task_due = EXCLUDED.task_due,
			mentions = EXCLUDED.mentions,
			comments = EXCLUDED.comments,
			status_changes = EXCLUDED.status_changes,
			member_added = EXCLUDED.member_added,
			updated_at = EXCLUDED.updated_at
		RETURNING updated_at
	`
	err := r.db.Pool.QueryRow(
		ctx,
		query,
		prefs.UserID,
		prefs.TaskAssigned,
		prefs.TaskDue,
		prefs.Mentions,
		prefs.Comments,
		prefs.StatusChanges,
		prefs.MemberAdded,
		now,
	).Scan(&prefs.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update notification preferences: %w", err)
	}
	return prefs, nil
}

// FindDueAndOverdueTasks returns tasks with due dates in the past or within 24 hours that are not DONE.
type DueTaskRecord struct {
	TaskID     uuid.UUID
	ProjectID  uuid.UUID
	Title      string
	AssignedTo uuid.UUID
	DueDate    time.Time
}

func (r *NotificationRepository) FindDueAndOverdueTasks(ctx context.Context) ([]DueTaskRecord, error) {
	query := `
		SELECT t.id, t.project_id, t.title, t.assigned_to, t.due_date
		FROM tasks t
		INNER JOIN project_members pm ON pm.project_id = t.project_id AND pm.user_id = t.assigned_to
		WHERE t.assigned_to IS NOT NULL
		  AND t.due_date IS NOT NULL
		  AND t.status <> 'DONE'
		  AND t.due_date <= (NOW() + INTERVAL '24 hours')
	`
	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DueTaskRecord
	for rows.Next() {
		var rec DueTaskRecord
		if err := rows.Scan(&rec.TaskID, &rec.ProjectID, &rec.Title, &rec.AssignedTo, &rec.DueDate); err == nil {
			results = append(results, rec)
		}
	}
	return results, nil
}
