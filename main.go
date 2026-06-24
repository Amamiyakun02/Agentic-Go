package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"AgenticGo/handlers"
	"AgenticGo/middlewares"
	"AgenticGo/services"
	_ "AgenticGo/docs" // Swagger docs

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title AgenticGo API
// @version 1.0
// @description Backend API for Personal Assistant (Golang Version)
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@aimer.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @host localhost:8081
// @BasePath /
func main() {
	// Load environment variables dari .env (jika ada)
	err := godotenv.Load()
	if err != nil {
		log.Println("[Warning] File .env tidak ditemukan, menggunakan environment system.")
	}

	// Inisialisasi Database
	if err := services.InitMongoDB(); err != nil {
		log.Fatalf("Gagal inisialisasi MongoDB: %v", err)
	}

	if err := services.InitRedis(); err != nil {
		log.Printf("Gagal inisialisasi Redis (opsional): %v", err)
	}

	// Membuat router
	r := gin.Default()

	// Setup CORS Middleware
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true, // Saat produksi, ganti dengan domain spesifik: AllowOrigins: []string{"https://admin.domain.com"}
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "api-key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Endpoint GET Home
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "AgenticGo Backend is running!",
		})
	})

	// Endpoint WebSocket Live Voice
	r.GET("/live-voice", handlers.LiveVoiceHandler)

	// Routing API Authentication
	auth := r.Group("/v1/auth")
	{
		auth.POST("/register", handlers.RegisterUser)
		auth.POST("/login", handlers.LoginUser)
	}

	// Routing API Admin (Dilindungi Middleware)
	admin := r.Group("/v1/admin")
	admin.Use(middlewares.RequireAuth())
	{
		admin.POST("/sync-device-apps", handlers.SyncDeviceApps)
		admin.POST("/sync-github", handlers.SyncGitHub)
	}

	// Swagger Endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	port := os.Getenv("PORT")
	if port == "" {
		port = "7860" // Port default Hugging Face Spaces
	}

	// Setup HTTP Server untuk Graceful Shutdown
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Jalankan server di goroutine
	go func() {
		log.Printf("Server berjalan di http://0.0.0.0:%s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Tunggu sinyal interupsi (SIGINT/SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Mematikan server secara perlahan (Graceful Shutdown)...")

	// Beri waktu maksimal 5 detik untuk menyelesaikan request berjalan
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown error:", err)
	}
	
	log.Println("Server berhasil dimatikan.")
}