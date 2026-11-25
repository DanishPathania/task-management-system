import {
  PaginatedTasks,
  Project,
  ProjectMember,
  Task,
  User,
  Comment,
  PaginatedActivityResponse,
  PaginatedNotifications,
  NotificationPreferences,
  TaskDependenciesResponse,
  TaskDependency,
  ProjectAnalytics,
  DashboardAnalytics,
  TaskAttachment,
  ProjectStatus,
  ProjectLabel,
  RecurringTask,
} from "@/types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

class ApiClient {
  private token: string | null = null;

  constructor() {
    if (typeof window !== "undefined") {
      this.token = sessionStorage.getItem("task_auth_token");
    }
  }

  setToken(token: string | null) {
    this.token = token;
    if (typeof window !== "undefined") {
      if (token) {
        sessionStorage.setItem("task_auth_token", token);
      } else {
        sessionStorage.removeItem("task_auth_token");
      }
    }
  }

  getToken(): string | null {
    if (!this.token && typeof window !== "undefined") {
      this.token = sessionStorage.getItem("task_auth_token");
    }
    return this.token;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${API_BASE_URL}${endpoint}`;
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...(options.headers as Record<string, string>),
    };

    const token = this.getToken();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }

    const response = await fetch(url, {
      ...options,
      headers,
      credentials: "include", // Send HttpOnly cookie
    });

    if (response.status === 204) {
      return {} as T;
    }

    let data;
    try {
      data = await response.json();
    } catch {
      data = null;
    }

    if (!response.ok) {
      const errorMsg = data?.message || data?.error || `Request failed with status ${response.status}`;
      throw new Error(errorMsg);
    }

    return data as T;
  }

  // Auth
  async register(name: string, email: string, password: string): Promise<{ message: string; user: User }> {
    return this.request<{ message: string; user: User }>("/auth/register", {
      method: "POST",
      body: JSON.stringify({ name, email, password }),
    });
  }

  async login(email: string, password: string): Promise<{ token: string; user: User }> {
    const res = await this.request<{ token: string; user: User }>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    });
    this.setToken(res.token);
    return res;
  }

  async logout(): Promise<void> {
    try {
      await this.request("/auth/logout", { method: "POST" });
    } finally {
      this.setToken(null);
    }
  }

  async getMe(): Promise<User> {
    return this.request<User>("/auth/me");
  }

  // Projects
  async listProjects(): Promise<Project[]> {
    return this.request<Project[]>("/projects");
  }

  async getProject(projectId: string): Promise<Project> {
    return this.request<Project>(`/projects/${projectId}`);
  }

  async createProject(name: string, description?: string): Promise<Project> {
    return this.request<Project>("/projects", {
      method: "POST",
      body: JSON.stringify({ name, description }),
    });
  }

  async updateProject(projectId: string, name?: string, description?: string): Promise<Project> {
    return this.request<Project>(`/projects/${projectId}`, {
      method: "PATCH",
      body: JSON.stringify({ name, description }),
    });
  }

  async deleteProject(projectId: string): Promise<void> {
    return this.request<void>(`/projects/${projectId}`, { method: "DELETE" });
  }

  // Project Members
  async listMembers(projectId: string): Promise<ProjectMember[]> {
    return this.request<ProjectMember[]>(`/projects/${projectId}/members`);
  }

  async addMember(projectId: string, email: string, role: string): Promise<ProjectMember> {
    return this.request<ProjectMember>(`/projects/${projectId}/members`, {
      method: "POST",
      body: JSON.stringify({ email, role }),
    });
  }

  async updateMemberRole(projectId: string, userId: string, role: string): Promise<ProjectMember> {
    return this.request<ProjectMember>(`/projects/${projectId}/members/${userId}`, {
      method: "PATCH",
      body: JSON.stringify({ role }),
    });
  }

  async removeMember(projectId: string, userId: string): Promise<void> {
    return this.request<void>(`/projects/${projectId}/members/${userId}`, { method: "DELETE" });
  }

  async getPresence(projectId: string): Promise<{ project_id: string; online_user_ids: string[] }> {
    return this.request<{ project_id: string; online_user_ids: string[] }>(`/projects/${projectId}/presence`);
  }

  // Tasks
  async listTasks(
    projectId: string,
    params?: {
      search?: string;
      status?: string;
      priority?: string;
      assignee?: string;
      label?: string;
      unassigned?: boolean;
      overdue?: boolean;
      due_before?: string;
      due_after?: string;
      include_subtasks?: boolean;
      sort_by?: string;
      page?: number;
      limit?: number;
    }
  ): Promise<PaginatedTasks> {
    const query = new URLSearchParams();
    if (params?.search) query.set("search", params.search);
    if (params?.status) query.set("status", params.status);
    if (params?.priority) query.set("priority", params.priority);
    if (params?.assignee) query.set("assignee", params.assignee);
    if (params?.label) query.set("label", params.label);
    if (params?.unassigned) query.set("unassigned", "true");
    if (params?.overdue) query.set("overdue", "true");
    if (params?.due_before) query.set("due_before", params.due_before);
    if (params?.due_after) query.set("due_after", params.due_after);
    if (params?.include_subtasks) query.set("include_subtasks", "true");
    if (params?.sort_by) query.set("sort_by", params.sort_by);
    if (params?.page) query.set("page", params.page.toString());
    if (params?.limit) query.set("limit", params.limit.toString());

    const qs = query.toString() ? `?${query.toString()}` : "";
    return this.request<PaginatedTasks>(`/projects/${projectId}/tasks${qs}`);
  }

  async listSubtasks(projectId: string, taskId: string): Promise<Task[]> {
    return this.request<Task[]>(`/projects/${projectId}/tasks/${taskId}/subtasks`);
  }

  async createSubtask(projectId: string, taskId: string, subtask: Partial<Task>): Promise<Task> {
    return this.request<Task>(`/projects/${projectId}/tasks/${taskId}/subtasks`, {
      method: "POST",
      body: JSON.stringify(subtask),
    });
  }

  async getTask(projectId: string, taskId: string): Promise<Task> {
    return this.request<Task>(`/projects/${projectId}/tasks/${taskId}`);
  }

  async createTask(projectId: string, task: Partial<Task>): Promise<Task> {
    return this.request<Task>(`/projects/${projectId}/tasks`, {
      method: "POST",
      body: JSON.stringify(task),
    });
  }

  async updateTask(projectId: string, taskId: string, updates: Partial<Task> & { unassign?: boolean; clear_start_date?: boolean; clear_due_date?: boolean }): Promise<Task> {
    return this.request<Task>(`/projects/${projectId}/tasks/${taskId}`, {
      method: "PATCH",
      body: JSON.stringify(updates),
    });
  }

  async deleteTask(projectId: string, taskId: string): Promise<void> {
    return this.request<void>(`/projects/${projectId}/tasks/${taskId}`, { method: "DELETE" });
  }

  // Comments
  async listComments(projectId: string, taskId: string): Promise<Comment[]> {
    return this.request<Comment[]>(`/projects/${projectId}/tasks/${taskId}/comments`);
  }

  async addComment(projectId: string, taskId: string, content: string): Promise<Comment> {
    return this.request<Comment>(`/projects/${projectId}/tasks/${taskId}/comments`, {
      method: "POST",
      body: JSON.stringify({ content }),
    });
  }

  async deleteComment(projectId: string, taskId: string, commentId: string): Promise<void> {
    return this.request<void>(`/projects/${projectId}/tasks/${taskId}/comments/${commentId}`, { method: "DELETE" });
  }

  // Activity Log
  async listActivity(projectId: string, page = 1, limit = 30): Promise<PaginatedActivityResponse> {
    return this.request<PaginatedActivityResponse>(`/projects/${projectId}/activity?page=${page}&limit=${limit}`);
  }

  // Notifications
  async listNotifications(unreadOnly = false, page = 1, limit = 20): Promise<PaginatedNotifications> {
    return this.request<PaginatedNotifications>(`/notifications?unread_only=${unreadOnly}&page=${page}&limit=${limit}`);
  }

  async markNotificationRead(notificationId: string): Promise<void> {
    return this.request<void>(`/notifications/${notificationId}/read`, { method: "PATCH" });
  }

  async markAllNotificationsRead(): Promise<void> {
    return this.request<void>("/notifications/read-all", { method: "POST" });
  }

  async getNotificationPreferences(): Promise<NotificationPreferences> {
    return this.request<NotificationPreferences>("/notifications/preferences");
  }

  async updateNotificationPreferences(prefs: Partial<NotificationPreferences>): Promise<NotificationPreferences> {
    return this.request<NotificationPreferences>("/notifications/preferences", {
      method: "PATCH",
      body: JSON.stringify(prefs),
    });
  }

  // Dependencies
  async listProjectDependencies(projectId: string): Promise<TaskDependency[]> {
    return this.request<TaskDependency[]>(`/projects/${projectId}/dependencies`);
  }

  async listTaskDependencies(projectId: string, taskId: string): Promise<TaskDependenciesResponse> {
    return this.request<TaskDependenciesResponse>(`/projects/${projectId}/tasks/${taskId}/dependencies`);
  }

  async addTaskDependency(projectId: string, taskId: string, dependsOnTaskId: string): Promise<TaskDependency> {
    return this.request<TaskDependency>(`/projects/${projectId}/tasks/${taskId}/dependencies`, {
      method: "POST",
      body: JSON.stringify({ depends_on_task_id: dependsOnTaskId }),
    });
  }

  async removeTaskDependency(projectId: string, taskId: string, dependencyId: string): Promise<void> {
    return this.request<void>(`/projects/${projectId}/tasks/${taskId}/dependencies/${dependencyId}`, {
      method: "DELETE",
    });
  }

  // Analytics
  async getProjectAnalytics(projectId: string): Promise<ProjectAnalytics> {
    return this.request<ProjectAnalytics>(`/projects/${projectId}/analytics`);
  }

  async getDashboardAnalytics(): Promise<DashboardAnalytics> {
    return this.request<DashboardAnalytics>("/analytics/dashboard");
  }

  // Attachments
  async listAttachments(projectId: string, taskId: string): Promise<TaskAttachment[]> {
    return this.request<TaskAttachment[]>(`/projects/${projectId}/tasks/${taskId}/attachments`);
  }

  async uploadAttachment(projectId: string, taskId: string, file: File): Promise<TaskAttachment> {
    const formData = new FormData();
    formData.append("file", file);
    const headers: Record<string, string> = {};
    const token = this.getToken();
    if (token) {
      headers["Authorization"] = `Bearer ${token}`;
    }
    const response = await fetch(`${API_BASE_URL}/projects/${projectId}/tasks/${taskId}/attachments`, {
      method: "POST",
      headers,
      body: formData,
      credentials: "include",
    });
    const data = await response.json().catch(() => null);
    if (!response.ok) {
      throw new Error(data?.message || data?.error || "Failed to upload file");
    }
    return data as TaskAttachment;
  }

  getAttachmentDownloadUrl(projectId: string, taskId: string, attachmentId: string): string {
    return `${API_BASE_URL}/projects/${projectId}/tasks/${taskId}/attachments/${attachmentId}/download`;
  }

  async deleteAttachment(projectId: string, taskId: string, attachmentId: string): Promise<void> {
    return this.request<void>(`/projects/${projectId}/tasks/${taskId}/attachments/${attachmentId}`, { method: "DELETE" });
  }

  // Custom Workflow & Labels
  async listStatuses(projectId: string): Promise<ProjectStatus[]> {
    return this.request<ProjectStatus[]>(`/projects/${projectId}/statuses`);
  }

  async createStatus(projectId: string, payload: { key: string; name: string; color: string; position: number; is_done: boolean; allowed_transitions: string[] }): Promise<ProjectStatus> {
    return this.request<ProjectStatus>(`/projects/${projectId}/statuses`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
  }

  async deleteStatus(projectId: string, statusId: string): Promise<void> {
    return this.request<void>(`/projects/${projectId}/statuses/${statusId}`, { method: "DELETE" });
  }

  async listLabels(projectId: string): Promise<ProjectLabel[]> {
    return this.request<ProjectLabel[]>(`/projects/${projectId}/labels`);
  }

  async createLabel(projectId: string, name: string, color: string): Promise<ProjectLabel> {
    return this.request<ProjectLabel>(`/projects/${projectId}/labels`, {
      method: "POST",
      body: JSON.stringify({ name, color }),
    });
  }

  async deleteLabel(projectId: string, labelId: string): Promise<void> {
    return this.request<void>(`/projects/${projectId}/labels/${labelId}`, { method: "DELETE" });
  }

  async listTaskLabels(projectId: string, taskId: string): Promise<ProjectLabel[]> {
    return this.request<ProjectLabel[]>(`/projects/${projectId}/tasks/${taskId}/labels`);
  }

  async attachTaskLabel(projectId: string, taskId: string, labelId: string): Promise<void> {
    return this.request<void>(`/projects/${projectId}/tasks/${taskId}/labels/${labelId}`, { method: "POST" });
  }

  async detachTaskLabel(projectId: string, taskId: string, labelId: string): Promise<void> {
    return this.request<void>(`/projects/${projectId}/tasks/${taskId}/labels/${labelId}`, { method: "DELETE" });
  }

  // Recurring Tasks
  async listRecurringTasks(projectId: string): Promise<RecurringTask[]> {
    return this.request<RecurringTask[]>(`/projects/${projectId}/recurring`);
  }

  async createRecurringTask(projectId: string, payload: Partial<RecurringTask>): Promise<RecurringTask> {
    return this.request<RecurringTask>(`/projects/${projectId}/recurring`, {
      method: "POST",
      body: JSON.stringify(payload),
    });
  }

  async toggleRecurringTask(projectId: string, recurringId: string, is_active: boolean): Promise<RecurringTask> {
    return this.request<RecurringTask>(`/projects/${projectId}/recurring/${recurringId}`, {
      method: "PATCH",
      body: JSON.stringify({ is_active }),
    });
  }

  async deleteRecurringTask(projectId: string, recurringId: string): Promise<void> {
    return this.request<void>(`/projects/${projectId}/recurring/${recurringId}`, { method: "DELETE" });
  }
}

export const api = new ApiClient();
