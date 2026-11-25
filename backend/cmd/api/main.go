package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/taskmgr/internal/config"
	"github.com/example/taskmgr/internal/database"
	"github.com/example/taskmgr/internal/handler"
	"github.com/example/taskmgr/internal/middleware"
	"github.com/example/taskmgr/internal/redis"
	"github.com/example/taskmgr/internal/repository"
	"github.com/example/taskmgr/internal/service"
	"github.com/example/taskmgr/internal/websocket"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	log.Println("Starting Real-Time Collaborative Task Management API Server...")

	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Database connection pool
	db, err := database.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxOpenConns, cfg.DBMaxIdleConns, cfg.DBConnMaxLifetime)
	if err != nil {
		log.Fatalf("Database connection failure: %v", err)
	}
	defer db.Close()

	// 3. Run database migrations
	if err := db.RunMigrations(ctx, "./migrations"); err != nil {
		log.Fatalf("Failed to execute database migrations: %v", err)
	}

	// 4. Redis connection
	redisClient, err := redis.Connect(ctx, cfg.RedisURL, cfg.RedisPassword)
	if err != nil {
		log.Printf("Redis connection warning: %v", err)
	}
	defer redisClient.Close()

	// 5. Repositories
	userRepo := repository.NewUserRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	activityRepo := repository.NewActivityRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	depRepo := repository.NewDependencyRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	attRepo := repository.NewAttachmentRepository(db)
	workflowRepo := repository.NewWorkflowRepository(db)
	recRepo := repository.NewRecurringRepository(db)

	// 6. Real-time WebSocket Hub
	hub := websocket.NewHub(projectRepo, redisClient)
	go hub.Run()

	// 7. Services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpirationHours)
	activityService := service.NewActivityService(activityRepo, projectRepo, userRepo, hub)
	notifService := service.NewNotificationService(notifRepo, projectRepo, hub)
	projectService := service.NewProjectService(projectRepo, userRepo, redisClient, hub, activityService, notifService)
	taskService := service.NewTaskService(taskRepo, projectRepo, workflowRepo, hub, activityService, notifService)
	commentService := service.NewCommentService(commentRepo, taskRepo, projectRepo, hub, activityService, notifService)
	depService := service.NewDependencyService(depRepo, projectRepo, hub, activityService)
	analyticsService := service.NewAnalyticsService(analyticsRepo, projectRepo)
	attService := service.NewAttachmentService(attRepo, taskRepo, projectRepo, hub, os.Getenv("UPLOAD_DIR"))
	workflowService := service.NewWorkflowService(workflowRepo, projectRepo, hub)
	recService := service.NewRecurringService(recRepo, projectRepo, hub)

	// Background due-date / overdue notification & recurring task scheduler
	go func() {
		notifService.CheckDueAndOverdueTasks(ctx)
		recService.ProcessDue(ctx)
		dueTicker := time.NewTicker(15 * time.Minute)
		recTicker := time.NewTicker(1 * time.Minute)
		defer dueTicker.Stop()
		defer recTicker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-dueTicker.C:
				notifService.CheckDueAndOverdueTasks(ctx)
			case <-recTicker.C:
				recService.ProcessDue(ctx)
			}
		}
	}()

	// 8. Handlers
	authHandler := handler.NewAuthHandler(authService, cfg)
	projectHandler := handler.NewProjectHandler(projectService, redisClient)
	taskHandler := handler.NewTaskHandler(taskService)
	commentHandler := handler.NewCommentHandler(commentService)
	activityHandler := handler.NewActivityHandler(activityService)
	notifHandler := handler.NewNotificationHandler(notifService)
	depHandler := handler.NewDependencyHandler(depService)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)
	attHandler := handler.NewAttachmentHandler(attService)
	workflowHandler := handler.NewWorkflowHandler(workflowService)
	recHandler := handler.NewRecurringHandler(recService)
	wsHandler := handler.NewWSHandler(hub, cfg, redisClient)

	// 9. HTTP Router
	r := chi.NewRouter()

	// Middlewares
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(60 * time.Second))

	// CORS Setup
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API Routes Group
	r.Route("/api/v1", func(api chi.Router) {
		// Health check (unauthenticated — used by Docker healthcheck and load balancers)
		api.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			handler.JSON(w, http.StatusOK, map[string]interface{}{
				"status":    "healthy",
				"timestamp": time.Now().UTC(),
				"service":   "taskmgr-api",
			})
		})

		// WebSocket endpoint
		api.Get("/ws", wsHandler.ServeHTTP)

		// Public Auth Routes — rate-limited to prevent brute-force attacks
		api.Route("/auth", func(authRouter chi.Router) {
			authRouter.With(middleware.RateLimitAuth).Post("/register", authHandler.Register)
			authRouter.With(middleware.RateLimitAuth).Post("/login", authHandler.Login)
			authRouter.Post("/logout", authHandler.Logout)
		})

		// Protected Routes
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))

			// Auth Me
			protected.Get("/auth/me", authHandler.Me)
			protected.Get("/analytics/dashboard", analyticsHandler.DashboardAnalytics)

			// Notifications
			protected.Route("/notifications", func(nr chi.Router) {
				nr.Get("/", notifHandler.List)
				nr.Post("/read-all", notifHandler.MarkAllRead)
				nr.Patch("/{notificationId}/read", notifHandler.MarkRead)
				nr.Get("/preferences", notifHandler.GetPreferences)
				nr.Patch("/preferences", notifHandler.UpdatePreferences)
			})

			// Projects
			protected.Route("/projects", func(pr chi.Router) {
				pr.Post("/", projectHandler.Create)
				pr.Get("/", projectHandler.List)

				pr.Route("/{projectId}", func(p chi.Router) {
					p.Get("/", projectHandler.Get)
					p.Patch("/", projectHandler.Update)
					p.Delete("/", projectHandler.Delete)
					p.Get("/presence", projectHandler.GetPresence)
					p.Get("/activity", activityHandler.List)
					p.Get("/analytics", analyticsHandler.ProjectAnalytics)
					p.Get("/dependencies", depHandler.ListForProject)

					// Custom Workflow & Labels
					p.Get("/statuses", workflowHandler.ListStatuses)
					p.Post("/statuses", workflowHandler.CreateStatus)
					p.Delete("/statuses/{statusId}", workflowHandler.DeleteStatus)
					p.Get("/labels", workflowHandler.ListLabels)
					p.Post("/labels", workflowHandler.CreateLabel)
					p.Delete("/labels/{labelId}", workflowHandler.DeleteLabel)

					// Recurring Tasks
					p.Get("/recurring", recHandler.List)
					p.Post("/recurring", recHandler.Create)
					p.Patch("/recurring/{recurringId}", recHandler.Toggle)
					p.Delete("/recurring/{recurringId}", recHandler.Delete)

					// Members
					p.Route("/members", func(mr chi.Router) {
						mr.Get("/", projectHandler.ListMembers)
						mr.Post("/", projectHandler.AddMember)
						mr.Patch("/{userId}", projectHandler.UpdateMemberRole)
						mr.Delete("/{userId}", projectHandler.RemoveMember)
					})

					// Tasks
					p.Route("/tasks", func(tr chi.Router) {
						tr.Post("/", taskHandler.Create)
						tr.Get("/", taskHandler.List)

						tr.Route("/{taskId}", func(t chi.Router) {
							t.Get("/", taskHandler.Get)
							t.Patch("/", taskHandler.Update)
							t.Delete("/", taskHandler.Delete)

							// Subtasks
							t.Get("/subtasks", taskHandler.ListSubtasks)
							t.Post("/subtasks", taskHandler.CreateSubtask)

							// Dependencies
							t.Get("/dependencies", depHandler.ListForTask)
							t.Post("/dependencies", depHandler.Add)
							t.Delete("/dependencies/{dependencyId}", depHandler.Remove)

							// Attachments
							t.Get("/attachments", attHandler.List)
							t.Post("/attachments", attHandler.Upload)
							t.Get("/attachments/{attachmentId}/download", attHandler.Download)
							t.Delete("/attachments/{attachmentId}", attHandler.Delete)

							// Task Labels
							t.Get("/labels", workflowHandler.ListTaskLabels)
							t.Post("/labels/{labelId}", workflowHandler.AttachLabel)
							t.Delete("/labels/{labelId}", workflowHandler.DetachLabel)

							// Comments
							t.Route("/comments", func(cr chi.Router) {
								cr.Post("/", commentHandler.Add)
								cr.Get("/", commentHandler.List)
								cr.Delete("/{commentId}", commentHandler.Delete)
							})
						})
					})
				})
			})
		})
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.ServerPort),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown channel
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server listening on port %s (env: %s)\n", cfg.ServerPort, cfg.ServerEnv)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to listen: %v", err)
		}
	}()

	<-shutdownChan
	log.Println("Shutting down API server gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	hub.Stop()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped cleanly")
}
