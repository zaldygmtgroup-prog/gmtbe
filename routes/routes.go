package routes

import (
	"net/http"
	"os"
	"strings"

	"begmt2/config"
	"begmt2/controllers"
	"begmt2/middleware"
	"begmt2/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(cfg config.Config, db *gorm.DB) *gin.Engine {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.Use(corsMiddleware(cfg))
	_ = os.MkdirAll(cfg.UploadDir, 0755)
	r.Static("/uploads", cfg.UploadDir)

	authController := controllers.NewAuthController(cfg, db)
	agentController := controllers.NewAgentController(cfg, db)
	productController := controllers.NewProductController(cfg, db)
	notificationHub := services.NewNotificationHub()
	preorderController := controllers.NewPreorderController(cfg, db, notificationHub)
	notificationController := controllers.NewNotificationController(db)
	onboardingController := controllers.NewAgentOnboardingController(db)
	pancakeController := controllers.NewPancakeController(cfg, db)
	marketingController := controllers.NewMarketingController(cfg, db)
	educationController := controllers.NewEducationController(cfg, db)
	knowledgeBaseController := controllers.NewKnowledgeBaseController(cfg, db)

	r.GET("/health", func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Failed to get database connection"})
			return
		}
		if err := sqlDB.Ping(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Database is unreachable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Server and database are healthy"})
	})

	// Pancake calls this URL directly. The shared secret is verified by the
	// controller because Pancake's webhook specification does not define a
	// cryptographic signature header.
	r.POST("/api/integrations/pancake/webhook", pancakeController.Webhook)

	pancake := r.Group("/api/pancake")
	pancake.Use(middleware.AuthMiddleware(cfg, db), middleware.RoleMiddleware("super_admin", "sales", "marketing"))
	{
		pancake.GET("/analytics", pancakeController.Analytics)
		pancake.POST("/conversions", pancakeController.UpsertConversion)
	}

	auth := r.Group("/api/auth")
	{
		auth.POST("/register", authController.Register)
		auth.POST("/register/google", authController.RegisterWithGoogle)
		auth.POST("/login", authController.Login)
		auth.POST("/google", authController.LoginWithGoogle)
		auth.POST("/forgot-password", authController.ForgotPassword)
		auth.POST("/verify-reset-token", authController.VerifyResetToken)
		auth.POST("/reset-password", authController.ResetPassword)
		auth.GET("/session", middleware.AuthMiddleware(cfg, db), authController.Session)
		auth.POST("/logout", middleware.AuthMiddleware(cfg, db), authController.Logout)
		auth.POST("/sso/code", middleware.AuthMiddleware(cfg, db), authController.CreateSSOCode)
		auth.POST("/sso/exchange", authController.ExchangeSSOCode)
		auth.GET("/me", middleware.AuthMiddleware(cfg, db), authController.Me)
		auth.POST("/apply-agent", middleware.AuthMiddleware(cfg, db), middleware.RoleMiddleware("user"), authController.ApplyAgent)
		auth.POST("/agent-verification", middleware.AuthMiddleware(cfg, db), middleware.RoleMiddleware("user"), authController.CompleteAgentVerification)
	}

	products := r.Group("/api/products")
	{
		products.GET("", productController.ListProducts)
		products.GET("/:id", productController.GetProduct)
		products.POST("", productController.CreateProduct)
		products.PUT("/:id", productController.UpdateProduct)
		products.DELETE("/:id", productController.DeleteProduct)
	}

	educations := r.Group("/api/educations")
	{
		// Public
		educations.GET("", educationController.ListEducations)
		educations.GET("/:id", educationController.GetEducation)

		// Protected CRUD (super_admin)
		educations.POST("", middleware.AuthMiddleware(cfg, db), middleware.RoleMiddleware("super_admin"), educationController.CreateEducation)
		educations.PUT("/:id", middleware.AuthMiddleware(cfg, db), middleware.RoleMiddleware("super_admin"), educationController.UpdateEducation)
		educations.DELETE("/:id", middleware.AuthMiddleware(cfg, db), middleware.RoleMiddleware("super_admin"), educationController.DeleteEducation)

		// Registration (user)
		educations.POST("/:id/register", middleware.AuthMiddleware(cfg, db), educationController.RegisterEducation)
	}

	preorders := r.Group("/api/preorders")
	preorders.Use(middleware.AuthMiddleware(cfg, db))
	{
		preorders.GET("", preorderController.ListPreorders)
		preorders.GET("/:id", preorderController.GetPreorder)
		preorders.POST("", middleware.RoleMiddleware("agent"), middleware.AgentStatusMiddleware(db, "official_agent"), preorderController.CreatePreorder)
		preorders.PUT("/:id", middleware.RoleMiddleware("agent"), middleware.AgentStatusMiddleware(db, "official_agent"), preorderController.UpdatePreorder)
		preorders.DELETE("/:id", middleware.RoleMiddleware("agent"), middleware.AgentStatusMiddleware(db, "official_agent"), preorderController.DeletePreorder)
		preorders.POST("/:id/submit", middleware.RoleMiddleware("agent"), middleware.AgentStatusMiddleware(db, "official_agent"), preorderController.SubmitPreorder)
		preorders.POST("/:id/payment-proof", middleware.RoleMiddleware("agent"), middleware.AgentStatusMiddleware(db, "official_agent"), preorderController.UploadPaymentProof)
		preorders.GET("/:id/pdf", middleware.RoleMiddleware("agent"), middleware.AgentStatusMiddleware(db, "official_agent"), preorderController.GetPreorderPDF)
	}

	superAdmin := r.Group("/api/super-admin")
	superAdmin.Use(middleware.AuthMiddleware(cfg, db), middleware.RoleMiddleware("super_admin"))
	{
		superAdmin.GET("/dashboard", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "super admin dashboard"})
		})
		superAdmin.GET("/agent-applications", authController.ListAgentApplications)
		superAdmin.PUT("/agent-applications/:id/status", authController.UpdateAgentApplicationStatus)
		superAdmin.GET("/withdraws", agentController.ListAllWithdraws)
		superAdmin.PUT("/withdraws/:id/approve", agentController.ApproveWithdraw)
		superAdmin.GET("/knowledge-base", knowledgeBaseController.GetAll)
		superAdmin.POST("/knowledge-base", knowledgeBaseController.SaveAll)
		superAdmin.GET("/knowledge-base/:roleKey", knowledgeBaseController.GetByRole)
	}

	agent := r.Group("/api/agent")
	agent.Use(middleware.AuthMiddleware(cfg, db), middleware.RoleMiddleware("agent"), middleware.AgentStatusMiddleware(db, "official_agent"))
	{
		agent.GET("/wallet", agentController.GetWallet)
		agent.POST("/commissions", agentController.CalculateCommission)
		agent.POST("/withdraws", agentController.CreateWithdraw)
		agent.GET("/withdraws", agentController.ListMyWithdraws)
		agent.GET("/preorders", preorderController.ListAgentPreorders)
		agent.GET("/preorders/stream", preorderController.StreamAgentPreorders)
	}

	agentOnboarding := r.Group("/api/agent/onboarding")
	agentOnboarding.Use(middleware.AuthMiddleware(cfg, db), middleware.RoleMiddleware("user", "agent"), middleware.AgentStatusMiddleware(db, "verif", "official_agent"))
	{
		agentOnboarding.GET("/videos", onboardingController.ListVideos)
		agentOnboarding.GET("/progress", onboardingController.GetProgress)
		agentOnboarding.POST("/progress", onboardingController.SaveProgress)
		agentOnboarding.DELETE("/progress", onboardingController.ResetProgress)
	}

	sales := r.Group("/api/sales")
	sales.Use(middleware.AuthMiddleware(cfg, db), middleware.RoleMiddleware("sales"))
	{
		sales.PUT("/preorders/:id/status", preorderController.UpdatePreorderStatus)
		sales.POST("/preorders/:id/payment-quotation", preorderController.SendPaymentQuotation)
		sales.POST("/preorders/:id/payment-proof", preorderController.UploadSalesPaymentProof)
		sales.GET("/notifications/stream", preorderController.StreamSalesNotifications)
	}

	notifications := r.Group("/api/notifications")
	notifications.Use(middleware.AuthMiddleware(cfg, db))
	{
		notifications.GET("", notificationController.ListNotifications)
		notifications.GET("/:id", notificationController.GetNotification)
		notifications.PUT("/:id/read", notificationController.MarkNotificationAsRead)
		notifications.PUT("/read-all", notificationController.MarkAllNotificationsAsRead)
	}

	marketing := r.Group("/api/marketing")
	{
		marketing.GET("/content-brief-cache", marketingController.GetCache)
		marketing.POST("/content-brief-cache", middleware.AuthMiddleware(cfg, db), middleware.RoleMiddleware("marketing", "super_admin"), marketingController.SaveCache)
		marketing.DELETE("/content-brief-cache", middleware.AuthMiddleware(cfg, db), middleware.RoleMiddleware("marketing", "super_admin"), marketingController.DeleteCache)
	}

	return r
}

func corsMiddleware(cfg config.Config) gin.HandlerFunc {
	allowedOrigins := make(map[string]bool)
	for _, origin := range cfg.CORSAllowedOrigins {
		origin = normalizeCORSOrigin(origin)
		if origin != "" {
			allowedOrigins[origin] = true
		}
	}

	return func(c *gin.Context) {
		origin := normalizeCORSOrigin(c.GetHeader("Origin"))
		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func normalizeCORSOrigin(origin string) string {
	return strings.TrimRight(strings.TrimSpace(origin), "/")
}
