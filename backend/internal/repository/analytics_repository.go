package repository

import (
	"context"

	"github.com/example/taskmgr/internal/database"
	"github.com/example/taskmgr/internal/models"
	"github.com/google/uuid"
)

type AnalyticsRepository struct {
	db *database.DB
}

func NewAnalyticsRepository(db *database.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) GetProjectAnalytics(ctx context.Context, projectID uuid.UUID) (*models.ProjectAnalytics, error) {
	analytics := &models.ProjectAnalytics{
		PriorityDistribution: make([]models.PriorityCount, 0),
		AssigneeBreakdown:    make([]models.MemberTaskCount, 0),
		RecentlyCompleted:    make([]*models.Task, 0),
		UpcomingDeadlines:    make([]*models.Task, 0),
		CompletionTrend:      make([]models.TrendPoint, 0),
	}

	// Core counts
	err := r.db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'DONE') AS completed,
			COUNT(*) FILTER (WHERE status = 'TODO') AS pending,
			COUNT(*) FILTER (WHERE status = 'IN_PROGRESS') AS in_progress,
			COUNT(*) FILTER (WHERE due_date IS NOT NULL AND due_date < NOW() AND status <> 'DONE') AS overdue
		FROM tasks
		WHERE project_id = $1 AND parent_task_id IS NULL
	`, projectID).Scan(
		&analytics.TotalTasks,
		&analytics.CompletedTasks,
		&analytics.PendingTasks,
		&analytics.InProgressTasks,
		&analytics.OverdueTasks,
	)
	if err != nil {
		return nil, err
	}

	if analytics.TotalTasks > 0 {
		analytics.CompletionPercent = int((analytics.CompletedTasks * 100) / analytics.TotalTasks)
	}

	// Priority distribution
	prioRows, err := r.db.Pool.Query(ctx, `
		SELECT priority, COUNT(*)
		FROM tasks
		WHERE project_id = $1 AND parent_task_id IS NULL
		GROUP BY priority
	`, projectID)
	if err == nil {
		defer prioRows.Close()
		for prioRows.Next() {
			var pc models.PriorityCount
			if prioRows.Scan(&pc.Priority, &pc.Count) == nil {
				analytics.PriorityDistribution = append(analytics.PriorityDistribution, pc)
			}
		}
	}

	// Assignee breakdown
	memRows, err := r.db.Pool.Query(ctx, `
		SELECT t.assigned_to, COALESCE(u.name, 'Unassigned') AS user_name,
		       COUNT(*) AS total,
		       COUNT(*) FILTER (WHERE t.status = 'DONE') AS done
		FROM tasks t
		LEFT JOIN users u ON t.assigned_to = u.id
		WHERE t.project_id = $1 AND t.parent_task_id IS NULL
		GROUP BY t.assigned_to, u.name
		ORDER BY total DESC
	`, projectID)
	if err == nil {
		defer memRows.Close()
		for memRows.Next() {
			var mc models.MemberTaskCount
			if memRows.Scan(&mc.UserID, &mc.UserName, &mc.Total, &mc.Done) == nil {
				analytics.AssigneeBreakdown = append(analytics.AssigneeBreakdown, mc)
			}
		}
	}

	// Recently completed tasks
	recentRows, err := r.db.Pool.Query(ctx, `
		SELECT id, project_id, title, status, priority, created_by, assigned_to, due_date, created_at, updated_at
		FROM tasks
		WHERE project_id = $1 AND parent_task_id IS NULL AND status = 'DONE'
		ORDER BY updated_at DESC
		LIMIT 5
	`, projectID)
	if err == nil {
		defer recentRows.Close()
		for recentRows.Next() {
			t := &models.Task{}
			if recentRows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Status, &t.Priority, &t.CreatedBy, &t.AssignedTo, &t.DueDate, &t.CreatedAt, &t.UpdatedAt) == nil {
				analytics.RecentlyCompleted = append(analytics.RecentlyCompleted, t)
			}
		}
	}

	// Upcoming deadlines
	upcomingRows, err := r.db.Pool.Query(ctx, `
		SELECT id, project_id, title, status, priority, created_by, assigned_to, due_date, created_at, updated_at
		FROM tasks
		WHERE project_id = $1 AND parent_task_id IS NULL AND status <> 'DONE' AND due_date IS NOT NULL
		ORDER BY due_date ASC
		LIMIT 5
	`, projectID)
	if err == nil {
		defer upcomingRows.Close()
		for upcomingRows.Next() {
			t := &models.Task{}
			if upcomingRows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Status, &t.Priority, &t.CreatedBy, &t.AssignedTo, &t.DueDate, &t.CreatedAt, &t.UpdatedAt) == nil {
				analytics.UpcomingDeadlines = append(analytics.UpcomingDeadlines, t)
			}
		}
	}

	// Completion trend over last 14 days
	trendRows, err := r.db.Pool.Query(ctx, `
		SELECT TO_CHAR(updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS day, COUNT(*)
		FROM tasks
		WHERE project_id = $1 AND status = 'DONE' AND updated_at >= NOW() - INTERVAL '14 days'
		GROUP BY day
		ORDER BY day ASC
	`, projectID)
	if err == nil {
		defer trendRows.Close()
		for trendRows.Next() {
			var tp models.TrendPoint
			if trendRows.Scan(&tp.Date, &tp.Count) == nil {
				analytics.CompletionTrend = append(analytics.CompletionTrend, tp)
			}
		}
	}

	return analytics, nil
}

func (r *AnalyticsRepository) GetDashboardAnalytics(ctx context.Context, userID uuid.UUID) (*models.DashboardAnalytics, error) {
	dash := &models.DashboardAnalytics{
		UpcomingDeadlines: make([]*models.Task, 0),
	}

	_ = r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM project_members WHERE user_id = $1`, userID).Scan(&dash.TotalProjects)

	err := r.db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE t.status = 'DONE') AS completed,
			COUNT(*) FILTER (WHERE t.status = 'TODO') AS pending,
			COUNT(*) FILTER (WHERE t.status = 'IN_PROGRESS') AS in_progress,
			COUNT(*) FILTER (WHERE t.due_date IS NOT NULL AND t.due_date < NOW() AND t.status <> 'DONE') AS overdue
		FROM tasks t
		INNER JOIN project_members pm ON pm.project_id = t.project_id AND pm.user_id = $1
		WHERE t.parent_task_id IS NULL
	`, userID).Scan(
		&dash.TotalTasks,
		&dash.CompletedTasks,
		&dash.PendingTasks,
		&dash.InProgressTasks,
		&dash.OverdueTasks,
	)
	if err != nil {
		return nil, err
	}

	if dash.TotalTasks > 0 {
		dash.CompletionPercent = int((dash.CompletedTasks * 100) / dash.TotalTasks)
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT t.id, t.project_id, t.title, t.status, t.priority, t.created_by, t.assigned_to, t.due_date, t.created_at, t.updated_at
		FROM tasks t
		INNER JOIN project_members pm ON pm.project_id = t.project_id AND pm.user_id = $1
		WHERE t.parent_task_id IS NULL AND t.status <> 'DONE' AND t.due_date IS NOT NULL
		ORDER BY t.due_date ASC
		LIMIT 6
	`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			t := &models.Task{}
			if rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Status, &t.Priority, &t.CreatedBy, &t.AssignedTo, &t.DueDate, &t.CreatedAt, &t.UpdatedAt) == nil {
				dash.UpcomingDeadlines = append(dash.UpcomingDeadlines, t)
			}
		}
	}

	return dash, nil
}
