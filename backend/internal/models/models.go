package models

import (
	"time"

	"github.com/google/uuid"
)

// Roles
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// Task Statuses
const (
	StatusTodo       = "TODO"
	StatusInProgress = "IN_PROGRESS"
	StatusDone       = "DONE"
)

// Task Priorities
const (
	PriorityLow    = "LOW"
	PriorityMedium = "MEDIUM"
	PriorityHigh   = "HIGH"
)

// Event Types for WebSockets and Redis Pub/Sub
const (
	EventTaskCreated     = "task.created"
	EventTaskUpdated     = "task.updated"
	EventTaskDeleted     = "task.deleted"
	EventTaskAssigned    = "task.assigned"
	EventTaskStatusChanged = "task.status_changed"
	EventCommentCreated  = "comment.created"
	EventCommentDeleted  = "comment.deleted"
	EventMemberAdded        = "project.member_added"
	EventMemberRemoved      = "project.member_removed"
	EventRoleChanged        = "project.role_changed"
	EventPresenceUpdated    = "presence.updated"
	EventActivityCreated    = "activity.created"
	EventNotificationCreated = "notification.created"
	EventSubtaskCreated     = "subtask.created"
	EventSubtaskUpdated     = "subtask.updated"
	EventSubtaskDeleted     = "subtask.deleted"
	EventDependencyChanged  = "task.dependency_changed"
	EventAttachmentChanged  = "task.attachment_changed"
	EventWorkflowChanged    = "project.workflow_changed"
)

// User represents an authenticated account
type User struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserResponse is the safe public representation of a user
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// Project represents a workspace/project
type Project struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedBy   uuid.UUID `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Role        string    `json:"user_role,omitempty"` // populated when fetched in context of a user
}

// ProjectMember represents membership of a user in a project
type ProjectMember struct {
	ProjectID uuid.UUID `json:"project_id"`
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
	UserName  string    `json:"user_name,omitempty"`
	UserEmail string    `json:"user_email,omitempty"`
}

// Task represents a task within a project
type Task struct {
	ID                    uuid.UUID  `json:"id"`
	ProjectID             uuid.UUID  `json:"project_id"`
	ParentTaskID          *uuid.UUID `json:"parent_task_id,omitempty"`
	Title                 string     `json:"title"`
	Description           string     `json:"description,omitempty"`
	Status                string     `json:"status"`
	Priority              string     `json:"priority"`
	CreatedBy             uuid.UUID  `json:"created_by"`
	CreatorName           string     `json:"creator_name,omitempty"`
	AssignedTo            *uuid.UUID `json:"assigned_to,omitempty"`
	AssigneeName          string     `json:"assignee_name,omitempty"`
	StartDate             *time.Time `json:"start_date,omitempty"`
	DueDate               *time.Time `json:"due_date,omitempty"`
	IsBlocked             bool       `json:"is_blocked"`
	SubtaskCount          int        `json:"subtask_count"`
	CompletedSubtaskCount int        `json:"completed_subtask_count"`
	CompletionPercent     int        `json:"completion_percent"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// Comment represents a comment on a task
type Comment struct {
	ID        uuid.UUID `json:"id"`
	TaskID    uuid.UUID `json:"task_id"`
	UserID    uuid.UUID `json:"user_id"`
	UserName  string    `json:"user_name,omitempty"`
	UserEmail string    `json:"user_email,omitempty"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Real-Time Event Envelope
type RealtimeEvent struct {
	Type         string      `json:"type"`      // always "event"
	Event        string      `json:"event"`     // e.g. "task.created"
	EventID      uuid.UUID   `json:"eventId"`
	ProjectID    uuid.UUID   `json:"projectId"`
	TargetUserID *uuid.UUID  `json:"targetUserId,omitempty"`
	Timestamp    time.Time   `json:"timestamp"`
	Payload      interface{} `json:"payload"`
}

// Pagination metadata
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

type PaginatedTasksResponse struct {
	Tasks      []*Task    `json:"tasks"`
	Pagination Pagination `json:"pagination"`
}

// Presence info for active user in a project
type UserPresence struct {
	UserID    uuid.UUID `json:"user_id"`
	UserName  string    `json:"user_name"`
	UserEmail string    `json:"user_email"`
	LastSeen  time.Time `json:"last_seen"`
	IsOnline  bool      `json:"is_online"`
}

// ActivityLog represents an immutable audit log entry for a project
type ActivityLog struct {
	ID          uuid.UUID   `json:"id"`
	ProjectID   uuid.UUID   `json:"project_id"`
	ActorID     *uuid.UUID  `json:"actor_id,omitempty"`
	ActorName   string      `json:"actor_name"`
	Action      string      `json:"action"`
	EntityType  string      `json:"entity_type"`
	EntityID    *uuid.UUID  `json:"entity_id,omitempty"`
	EntityTitle string      `json:"entity_title"`
	OldValue    interface{} `json:"old_value,omitempty"`
	NewValue    interface{} `json:"new_value,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

type PaginatedActivityResponse struct {
	Activities []*ActivityLog `json:"activities"`
	Pagination Pagination     `json:"pagination"`
}

// Notification represents a persistent in-app notification for a user
type Notification struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	ProjectID *uuid.UUID `json:"project_id,omitempty"`
	TaskID    *uuid.UUID `json:"task_id,omitempty"`
	ActorID   *uuid.UUID `json:"actor_id,omitempty"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	IsRead    bool       `json:"is_read"`
	CreatedAt time.Time  `json:"created_at"`
}

// NotificationPreferences stores per-user notification toggles
type NotificationPreferences struct {
	UserID        uuid.UUID `json:"user_id"`
	TaskAssigned  bool      `json:"task_assigned"`
	TaskDue       bool      `json:"task_due"`
	Mentions      bool      `json:"mentions"`
	Comments      bool      `json:"comments"`
	StatusChanges bool      `json:"status_changes"`
	MemberAdded   bool      `json:"member_added"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TaskDependency represents a Finish-to-Start prerequisite relationship
type TaskDependency struct {
	ID              uuid.UUID `json:"id"`
	ProjectID       uuid.UUID `json:"project_id"`
	TaskID          uuid.UUID `json:"task_id"`
	DependsOnTaskID uuid.UUID `json:"depends_on_task_id"`
	DependencyType  string    `json:"dependency_type"`
	DependsOnTitle  string    `json:"depends_on_title,omitempty"`
	DependsOnStatus string    `json:"depends_on_status,omitempty"`
	TaskTitle       string    `json:"task_title,omitempty"`
	TaskStatus      string    `json:"task_status,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type TaskDependenciesResponse struct {
	Prerequisites []*TaskDependency `json:"prerequisites"`
	Dependents    []*TaskDependency `json:"dependents"`
}

// Analytics models
type PriorityCount struct {
	Priority string `json:"priority"`
	Count    int64  `json:"count"`
}

type MemberTaskCount struct {
	UserID   *uuid.UUID `json:"user_id,omitempty"`
	UserName string     `json:"user_name"`
	Total    int64      `json:"total"`
	Done     int64      `json:"done"`
}

type TrendPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type ProjectAnalytics struct {
	TotalTasks           int64             `json:"total_tasks"`
	CompletedTasks       int64             `json:"completed_tasks"`
	PendingTasks         int64             `json:"pending_tasks"`
	InProgressTasks      int64             `json:"in_progress_tasks"`
	OverdueTasks         int64             `json:"overdue_tasks"`
	CompletionPercent    int               `json:"completion_percent"`
	PriorityDistribution []PriorityCount   `json:"priority_distribution"`
	AssigneeBreakdown    []MemberTaskCount `json:"assignee_breakdown"`
	RecentlyCompleted    []*Task           `json:"recently_completed"`
	UpcomingDeadlines    []*Task           `json:"upcoming_deadlines"`
	CompletionTrend      []TrendPoint      `json:"completion_trend"`
}

type DashboardAnalytics struct {
	TotalProjects     int64           `json:"total_projects"`
	TotalTasks        int64           `json:"total_tasks"`
	CompletedTasks    int64           `json:"completed_tasks"`
	PendingTasks      int64           `json:"pending_tasks"`
	InProgressTasks   int64           `json:"in_progress_tasks"`
	OverdueTasks      int64           `json:"overdue_tasks"`
	CompletionPercent int             `json:"completion_percent"`
	UpcomingDeadlines []*Task         `json:"upcoming_deadlines"`
}

// TaskAttachment represents a file attached to a task
type TaskAttachment struct {
	ID           uuid.UUID  `json:"id"`
	ProjectID    uuid.UUID  `json:"project_id"`
	TaskID       uuid.UUID  `json:"task_id"`
	UploadedBy   *uuid.UUID `json:"uploaded_by,omitempty"`
	OriginalName string     `json:"original_name"`
	StorageKey   string     `json:"-"`
	ContentType  string     `json:"content_type"`
	SizeBytes    int64      `json:"size_bytes"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ProjectStatus represents a configurable workflow column/status
type ProjectStatus struct {
	ID                 uuid.UUID `json:"id"`
	ProjectID          uuid.UUID `json:"project_id"`
	Key                string    `json:"key"`
	Name               string    `json:"name"`
	Color              string    `json:"color"`
	Position           int       `json:"position"`
	IsDone             bool      `json:"is_done"`
	AllowedTransitions []string  `json:"allowed_transitions"`
	CreatedAt          time.Time `json:"created_at"`
}

// ProjectLabel represents a custom task label
type ProjectLabel struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

// RecurringTask represents a recurring task series
type RecurringTask struct {
	ID            uuid.UUID  `json:"id"`
	ProjectID     uuid.UUID  `json:"project_id"`
	CreatedBy     uuid.UUID  `json:"created_by"`
	Title         string     `json:"title"`
	Description   string     `json:"description,omitempty"`
	Priority      string     `json:"priority"`
	AssignedTo    *uuid.UUID `json:"assigned_to,omitempty"`
	Frequency     string     `json:"frequency"`
	IntervalValue int        `json:"interval_value"`
	StartDate     time.Time  `json:"start_date"`
	EndDate       *time.Time `json:"end_date,omitempty"`
	NextRunAt     time.Time  `json:"next_run_at"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
