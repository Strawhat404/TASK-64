package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"compliance-console/internal/handlers"
	"compliance-console/internal/middleware"
	"compliance-console/internal/services"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

func main() {
	// Build database connection string
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := getEnvOrDefault("DB_HOST", "localhost")
		port := getEnvOrDefault("DB_PORT", "5432")
		user := getEnvOrDefault("DB_USER", "postgres")
		password := getEnvOrDefault("DB_PASSWORD", "postgres")
		dbName := getEnvOrDefault("DB_NAME", "compliance_console")
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			host, port, user, password, dbName)
	}

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	log.Println("Connected to PostgreSQL database")

	// Initialize Echo
	e := echo.New()
	e.HideBanner = true

	// Global middleware
	e.Use(echomw.Logger())
	e.Use(echomw.Recover())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Captcha-Token"},
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// Initialize services
	encryptionService := services.NewEncryptionService()
	auditLedger := services.NewAuditLedger(db)
	_ = auditLedger // used in security handler

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db)
	userHandler := handlers.NewUserHandler(db)
	serviceHandler := handlers.NewServiceHandler(db)
	scheduleHandler := handlers.NewScheduleHandler(db)
	staffHandler := handlers.NewStaffHandler(db)
	auditHandler := handlers.NewAuditHandler(db)
	governanceHandler := handlers.NewGovernanceHandler(db)
	reconciliationHandler := handlers.NewReconciliationHandler(db)
	securityHandler := handlers.NewSecurityHandler(db, encryptionService)

	// API route group with rate limiting (300 req/min per user/IP)
	api := e.Group("/api")

	// Health check
	api.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.POST("/login", authHandler.Login, middleware.RateLimiter(db, 10, 1), middleware.CaptchaCheck(db))
	auth.POST("/logout", authHandler.Logout, middleware.AuthMiddleware(db))
	auth.GET("/session", authHandler.GetSession, middleware.AuthMiddleware(db))

	// Protected routes
	protected := api.Group("", middleware.AuthMiddleware(db), middleware.RateLimiter(db, 300, 1))

	// User management (admin only)
	users := protected.Group("/users", middleware.RoleGuard("Administrator"))
	users.GET("", userHandler.ListUsers)
	users.GET("/:id", userHandler.GetUser)
	users.POST("", userHandler.CreateUser)
	users.PUT("/:id", userHandler.UpdateUser)
	users.DELETE("/:id", userHandler.DeactivateUser)

	// Service catalog
	svc := protected.Group("/services")
	svc.GET("", serviceHandler.ListServices)
	svc.GET("/:id", serviceHandler.GetService)
	svc.GET("/:id/pricing", serviceHandler.GetPricing)
	svc.POST("", serviceHandler.CreateService, middleware.RoleGuard("Administrator", "Scheduler"))
	svc.PUT("/:id", serviceHandler.UpdateService, middleware.RoleGuard("Administrator", "Scheduler"))
	svc.DELETE("/:id", serviceHandler.DeleteService, middleware.RoleGuard("Administrator"))

	// Schedules
	sched := protected.Group("/schedules")
	sched.GET("", scheduleHandler.ListSchedules)
	sched.POST("", scheduleHandler.CreateSchedule, middleware.RoleGuard("Administrator", "Scheduler"))
	sched.PUT("/:id", scheduleHandler.UpdateSchedule, middleware.RoleGuard("Administrator", "Scheduler"))
	sched.DELETE("/:id", scheduleHandler.CancelSchedule, middleware.RoleGuard("Administrator", "Scheduler"))
	sched.POST("/:id/confirm", scheduleHandler.ConfirmAssignment)
	sched.GET("/available-staff", scheduleHandler.FindAvailableStaff)
	sched.POST("/:id/backup", scheduleHandler.RequestBackup, middleware.RoleGuard("Administrator", "Scheduler"))
	sched.POST("/backup/:id/confirm", scheduleHandler.ConfirmBackup, middleware.RoleGuard("Administrator", "Scheduler"))

	// Staff roster
	staff := protected.Group("/staff")
	staff.GET("", staffHandler.ListStaff)
	staff.GET("/:id", staffHandler.GetStaff)
	staff.POST("", staffHandler.CreateStaff, middleware.RoleGuard("Administrator", "Scheduler"))
	staff.PUT("/:id", staffHandler.UpdateStaff, middleware.RoleGuard("Administrator", "Scheduler"))
	staff.DELETE("/:id", staffHandler.DeleteStaff, middleware.RoleGuard("Administrator"))
	staff.GET("/:id/credentials", staffHandler.ListCredentials)
	staff.POST("/:id/credentials", staffHandler.AddCredential, middleware.RoleGuard("Administrator", "Scheduler"))
	staff.GET("/:id/availability", staffHandler.ListAvailability)
	staff.POST("/:id/availability", staffHandler.AddAvailability, middleware.RoleGuard("Administrator", "Scheduler"))

	// Audit logs (auditor only)
	audit := protected.Group("/audit", middleware.RoleGuard("Auditor", "Administrator"))
	audit.GET("/logs", auditHandler.ListAuditLogs)

	// Content Governance routes
	gov := protected.Group("/governance")
	gov.POST("/content", governanceHandler.CreateContent, middleware.RoleGuard("Administrator", "Reviewer"))
	gov.GET("/content", governanceHandler.ListContent)
	gov.GET("/content/:id", governanceHandler.GetContent)
	gov.PUT("/content/:id", governanceHandler.UpdateContent, middleware.RoleGuard("Administrator", "Reviewer"))
	gov.POST("/content/:id/submit", governanceHandler.SubmitForReview, middleware.RoleGuard("Administrator", "Reviewer"))
	gov.POST("/content/:id/promote", governanceHandler.PromoteContent, middleware.RoleGuard("Administrator"))
	gov.GET("/content/:id/versions", governanceHandler.GetVersionHistory)
	gov.POST("/content/:id/rollback", governanceHandler.RollbackContent, middleware.RoleGuard("Administrator"))
	gov.GET("/reviews/pending", governanceHandler.ListPendingReviews, middleware.RoleGuard("Administrator", "Reviewer"))
	gov.POST("/reviews/:id/decide", governanceHandler.ReviewDecision, middleware.RoleGuard("Administrator", "Reviewer"))
	gov.GET("/gray-release", governanceHandler.GetGrayReleaseItems)
	gov.GET("/rules", governanceHandler.ListRules, middleware.RoleGuard("Administrator"))
	gov.POST("/rules", governanceHandler.CreateRule, middleware.RoleGuard("Administrator"))
	gov.PUT("/rules/:id", governanceHandler.UpdateRule, middleware.RoleGuard("Administrator"))
	gov.DELETE("/rules/:id", governanceHandler.DeleteRule, middleware.RoleGuard("Administrator"))
	gov.GET("/content/:id/versions/diff", governanceHandler.DiffVersions, middleware.RoleGuard("Administrator", "Reviewer"))
	gov.POST("/relationships", governanceHandler.CreateRelationship, middleware.RoleGuard("Administrator", "Reviewer"))
	gov.GET("/relationships", governanceHandler.ListRelationships)
	gov.DELETE("/relationships/:id", governanceHandler.DeleteRelationship, middleware.RoleGuard("Administrator"))
	gov.POST("/content/:id/re-review", governanceHandler.ReReview, middleware.RoleGuard("Administrator", "Reviewer"))

	// Financial Reconciliation routes
	recon := protected.Group("/reconciliation", middleware.RoleGuard("Administrator", "Auditor"))
	recon.POST("/import", reconciliationHandler.ImportFeed)
	recon.GET("/feeds", reconciliationHandler.ListFeeds)
	recon.GET("/feeds/:id", reconciliationHandler.GetFeed)
	recon.POST("/feeds/:id/match", reconciliationHandler.RunMatching)
	recon.GET("/matches", reconciliationHandler.GetMatchResults)
	recon.GET("/exceptions", reconciliationHandler.ListExceptions)
	recon.GET("/exceptions/export", reconciliationHandler.ExportExceptions)
	recon.GET("/exceptions/:id", reconciliationHandler.GetException)
	recon.PUT("/exceptions/:id/assign", reconciliationHandler.AssignException)
	recon.PUT("/exceptions/:id/resolve", reconciliationHandler.ResolveException)
	recon.GET("/summary", reconciliationHandler.GetSummary)

	// Security & Data Lifecycle routes
	sec := protected.Group("/security", middleware.RoleGuard("Administrator"))
	sec.POST("/sensitive", securityHandler.StoreSensitiveData)
	sec.GET("/sensitive", securityHandler.ListSensitiveData)
	sec.GET("/sensitive/:id", securityHandler.GetSensitiveData)
	sec.POST("/sensitive/:id/reveal", securityHandler.RevealSensitiveData)
	sec.DELETE("/sensitive/:id", securityHandler.DeleteSensitiveData)
	sec.POST("/keys/rotate", securityHandler.RotateEncryptionKey)
	sec.GET("/keys", securityHandler.GetKeyStatus)
	sec.GET("/keys/rotation-due", securityHandler.GetRotationDue)
	sec.GET("/audit-ledger", securityHandler.GetAuditLedger)
	sec.POST("/audit-ledger/verify", securityHandler.VerifyAuditChain)
	sec.GET("/retention", securityHandler.GetRetentionPolicies)
	sec.POST("/retention/cleanup", securityHandler.RunRetentionCleanup)
	sec.GET("/rate-limits", securityHandler.GetRateLimitStatus)
	sec.POST("/legal-holds", securityHandler.CreateLegalHold)
	sec.GET("/legal-holds", securityHandler.ListLegalHolds)
	sec.PUT("/legal-holds/:id/release", securityHandler.ReleaseLegalHold)

	// Start server with TLS if certificates are available
	certFile := getEnvOrDefault("TLS_CERT_FILE", "certs/server.crt")
	keyFile := getEnvOrDefault("TLS_KEY_FILE", "certs/server.key")
	useTLS := false

	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			useTLS = true
		}
	}

	go func() {
		if useTLS {
			log.Printf("Starting server with TLS on :8443 (cert: %s, key: %s)", certFile, keyFile)
			if err := e.StartTLS(":8443", certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("TLS Server error: %v", err)
			}
		} else {
			log.Println("TLS certificates not found, starting without TLS on :8080")
			if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		}
	}()

	if useTLS {
		log.Println("Server started on :8443 (TLS)")
	} else {
		log.Println("Server started on :8080 (HTTP)")
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
