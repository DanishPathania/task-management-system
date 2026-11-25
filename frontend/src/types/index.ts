export type TaskStatus = string;
export type TaskPriority = 'LOW' | 'MEDIUM' | 'HIGH';
export type ProjectRole = 'admin' | 'member';

export interface User {
  id: string;
  name: string;
  email: string;
  created_at: string;
  updated_at: string;
}

export interface Project {
  id: string;
  name: string;
  description?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  user_role?: ProjectRole;
}

export interface ProjectMember {
  project_id: string;
  user_id: string;
  role: ProjectRole;
  joined_at: string;
  user_name?: string;
  user_email?: string;
}

export interface Task {
  id: string;
  project_id: string;
  parent_task_id?: string;
  title: string;
  description?: string;
  status: TaskStatus;
  priority: TaskPriority;
  created_by: string;
  creator_name?: string;
  assigned_to?: string;
  assignee_name?: string;
  start_date?: string;
  due_date?: string;
  is_blocked?: boolean;
  subtask_count?: number;
  completed_subtask_count?: number;
  completion_percent?: number;
  created_at: string;
  updated_at: string;
}

export interface Comment {
  id: string;
  task_id: string;
  user_id: string;
  user_name?: string;
  user_email?: string;
  content: string;
  created_at: string;
  updated_at: string;
}

export interface Pagination {
  page: number;
  limit: number;
  total_items: number;
  total_pages: number;
}

export interface PaginatedTasks {
  tasks: Task[];
  pagination: Pagination;
}

export interface RealtimeEvent<T = any> {
  type: string;
  event: string;
  eventId: string;
  projectId: string;
  timestamp: string;
  payload: T;
}

export interface OnlineUserPresence {
  user_id: string;
  is_online: boolean;
}

export interface ActivityLog {
  id: string;
  project_id: string;
  actor_id?: string;
  actor_name: string;
  action: string;
  entity_type: string;
  entity_id?: string;
  entity_title: string;
  old_value?: any;
  new_value?: any;
  created_at: string;
}

export interface PaginatedActivityResponse {
  activities: ActivityLog[];
  pagination: Pagination;
}

export interface Notification {
  id: string;
  user_id: string;
  project_id?: string;
  task_id?: string;
  actor_id?: string;
  type: string;
  title: string;
  message: string;
  is_read: boolean;
  created_at: string;
}

export interface PaginatedNotifications {
  notifications: Notification[];
  unread_count: number;
  pagination: Pagination;
}

export interface NotificationPreferences {
  user_id: string;
  task_assigned: boolean;
  task_due: boolean;
  mentions: boolean;
  comments: boolean;
  status_changes: boolean;
  member_added: boolean;
  updated_at: string;
}

export interface TaskDependency {
  id: string;
  project_id: string;
  task_id: string;
  depends_on_task_id: string;
  dependency_type: string;
  depends_on_title?: string;
  depends_on_status?: string;
  task_title?: string;
  task_status?: string;
  created_at: string;
}

export interface TaskDependenciesResponse {
  prerequisites: TaskDependency[];
  dependents: TaskDependency[];
}

export interface ProjectAnalytics {
  total_tasks: number;
  completed_tasks: number;
  pending_tasks: number;
  in_progress_tasks: number;
  overdue_tasks: number;
  completion_percent: number;
  priority_distribution: { priority: string; count: number }[];
  assignee_breakdown: { user_id?: string; user_name: string; total: number; done: number }[];
  recently_completed: Task[];
  upcoming_deadlines: Task[];
  completion_trend: { date: string; count: number }[];
}

export interface DashboardAnalytics {
  total_projects: number;
  total_tasks: number;
  completed_tasks: number;
  pending_tasks: number;
  in_progress_tasks: number;
  overdue_tasks: number;
  completion_percent: number;
  upcoming_deadlines: Task[];
}

export interface TaskAttachment {
  id: string;
  project_id: string;
  task_id: string;
  uploaded_by?: string;
  original_name: string;
  content_type: string;
  size_bytes: number;
  created_at: string;
}

export interface ProjectStatus {
  id: string;
  project_id: string;
  key: string;
  name: string;
  color: string;
  position: number;
  is_done: boolean;
  allowed_transitions: string[];
  created_at: string;
}

export interface ProjectLabel {
  id: string;
  project_id: string;
  name: string;
  color: string;
  created_at: string;
}

export interface RecurringTask {
  id: string;
  project_id: string;
  created_by: string;
  title: string;
  description?: string;
  priority: TaskPriority;
  assigned_to?: string;
  frequency: 'DAILY' | 'WEEKLY' | 'MONTHLY';
  interval_value: number;
  start_date: string;
  end_date?: string;
  next_run_at: string;
  last_run_at?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}
