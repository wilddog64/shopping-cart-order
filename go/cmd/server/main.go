package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wilddog64/shopping-cart-order/internal/auth"
	"github.com/wilddog64/shopping-cart-order/internal/checkout"
	"github.com/wilddog64/shopping-cart-order/internal/config"
	"github.com/wilddog64/shopping-cart-order/internal/events"
	"github.com/wilddog64/shopping-cart-order/internal/health"
	"github.com/wilddog64/shopping-cart-order/internal/httpx"
	"github.com/wilddog64/shopping-cart-order/internal/order"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const version = "0.1.0"

func main() {
	cfg := config.Load()
	logger := newLogger()
	defer func() {
		_ = logger.Sync()
	}()

	store, err := order.NewPostgresStore(context.Background(), cfg.DatabaseURI())
	if err != nil {
		logger.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer store.Close()

	publisher := events.NewRabbitPublisher(cfg.RabbitMQURI(), logger)
	orderService := order.NewService(store, publisher, logger)
	orderHandler := order.NewHandler(orderService, logger)
	checkoutHandler := checkout.NewHandler(orderService, checkout.NewBasketClient(cfg.BasketServiceURL), checkout.NewPaymentClient(cfg.PaymentServiceURL), cfg.PaymentGateway, logger)
	healthHandler := health.NewHandler(store, version)
	rateLimiter := httpx.NewRateLimiter(cfg.RateLimitPerSecond, cfg.RateLimitBurst)

	router := gin.New()
	gin.SetMode(gin.ReleaseMode)
	router.Use(gin.Recovery())
	router.Use(httpx.SecurityHeaders())
	router.Use(httpx.CorrelationID())
	router.Use(httpx.RequestLogger(logger))

	router.GET("/actuator/health", healthHandler.Health)
	router.GET("/actuator/health/liveness", healthHandler.Liveness)
	router.GET("/actuator/health/readiness", healthHandler.Readiness)
	router.GET("/actuator/info", healthHandler.Info)
	router.GET("/actuator/prometheus", gin.WrapH(promhttp.Handler()))

	var authMiddleware gin.HandlerFunc
	if cfg.OAuth2Enabled {
		validator := auth.NewJWTValidator(cfg.OAuth2IssuerURI, cfg.OAuth2ClientID, logger)
		authMiddleware = httpx.AuthMiddleware(validator, logger)
	} else {
		authMiddleware = httpx.MockAuthMiddleware()
	}

	api := router.Group("/api/orders")
	api.Use(rateLimiter.Middleware())
	api.Use(authMiddleware)
	{
		api.POST("", orderHandler.CreateOrder)
		api.POST("/checkout", checkoutHandler.Checkout)
		api.GET("", orderHandler.ListOrdersByCustomer)
		api.GET("/:orderId", orderHandler.GetOrder)
		api.PATCH("/:orderId/status", orderHandler.UpdateOrderStatus)
		api.POST("/:orderId/cancel", orderHandler.CancelOrder)
	}

	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("starting order service", zap.String("version", version), zap.String("port", cfg.ServerPort))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("failed to shutdown server", zap.Error(err))
	}

	if err := publisher.Close(); err != nil {
		logger.Error("failed to close publisher", zap.Error(err))
	}
}

func newLogger() *zap.Logger {
	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Development:      false,
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}

	logger, err := cfg.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	return logger
}
