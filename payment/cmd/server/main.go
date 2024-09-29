package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"context"

	_ "github.com/VuThanhThien/golang-gorm-postgres/payment/docs"
	"github.com/VuThanhThien/golang-gorm-postgres/payment/internal/initializers"
	"github.com/VuThanhThien/golang-gorm-postgres/payment/internal/middleware"
	"github.com/VuThanhThien/golang-gorm-postgres/payment/internal/routes"
	"github.com/VuThanhThien/golang-gorm-postgres/payment/pkg/logger"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/streadway/amqp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var (
	server *gin.Engine
	RMQ    *amqp.Connection
)

func init() {
	log := logger.NewLogger()
	config, err := initializers.LoadConfig(".")
	if err != nil {
		log.Error().Err(err).Msg("Could not load environment variables")
	}

	RMQ = initializers.ConnectRMQ(&config, context.Background())
	initializers.InitRedis(&config)
	initializers.ConnectDB(&config)
	if config.EnableAutoMigrate == "true" {
		if err := initializers.Migrate(); err != nil {
			log.Error().Err(err).Msg("Failed to run database migrations")
		}
	}

	server = gin.Default()
}

//	@title						Swagger Payment API
//	@version					1.0
//	@description				This is payment service golang server.
//	@host						localhost:8003
//	@BasePath					/api
//	@securityDefinitions.basic	BasicAuth
//	@securityDefinitions.apikey	Bearer
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space and JWT token.

func main() {
	log := logger.NewLogger()

	config, err := initializers.LoadConfig(".")
	if err != nil {
		log.Error().Err(err).Msg("Could not load environment variables")
	}

	runGinServer(config, log)

}

func runGinServer(config initializers.Config, log zerolog.Logger) {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:" + config.ServerPort, config.ClientOrigin}
	corsConfig.AllowCredentials = true

	// To implement a graceful shutdown while using Gin, you should create a http.Server
	// instance yourself and avoid using server.Run()
	srv := &http.Server{
		Addr:    ":" + config.ServerPort,
		Handler: server.Handler(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("Listen")
		}
	}()

	server.Use(cors.New(corsConfig))
	server.Use(middleware.RequestIDMiddleware())
	server.Use(middleware.LoggingMiddleware())
	server.Use(gzip.Gzip(gzip.DefaultCompression))
	server.Use(middleware.RateLimiter())
	// add timestamp to name to avoid overwrite this log
	f, _ := os.Create("tmp/gin.log")
	gin.DefaultWriter = io.MultiWriter(f)
	server.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s - [%s] \"%s: %s %s %d\" %s %s\n",
			param.TimeStamp.Format(time.DateTime),
			param.ClientIP,
			param.Keys[middleware.OperationIDKey],
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.ErrorMessage,
		)
	}))
	server.Use(middleware.Recover())
	server.LoadHTMLGlob("./public/html/*")
	server.Static("/public", "./public")
	server.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	server.NoRoute(func(c *gin.Context) {
		c.HTML(404, "404.html", gin.H{})
	})

	routes.SetupRoutes(server, initializers.DB, RMQ, log, &config)

	fmt.Printf("🚀 ~running on: http://localhost:%s/swagger/index.html 🚀 \n", config.ServerPort)

	// Wait for interrupt signal to gracefully shut down the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

}
