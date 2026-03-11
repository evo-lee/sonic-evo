package main

import (
	"context"
	"time"

	"go.uber.org/fx"

	"github.com/go-sonic/sonic/cache"
	"github.com/go-sonic/sonic/config"
	"github.com/go-sonic/sonic/dal"
	"github.com/go-sonic/sonic/event"
	"github.com/go-sonic/sonic/event/listener"
	"github.com/go-sonic/sonic/handler"
	"github.com/go-sonic/sonic/handler/middleware"
	"github.com/go-sonic/sonic/injection"
	"github.com/go-sonic/sonic/log"
	"github.com/go-sonic/sonic/template"
	"github.com/go-sonic/sonic/template/extension"
)

// NewLoginRateLimitMiddleware creates a rate limiter for login endpoints
func NewLoginRateLimitMiddleware() *middleware.RateLimitMiddleware {
	return middleware.NewRateLimitMiddleware(middleware.RateLimitConfig{
		RequestsPerWindow: 5,              // 5 requests
		Window:            time.Minute,     // per minute
		MaxBurst:          5,               // allow burst of 5
	})
}

// NewTimeoutMiddleware creates a timeout middleware for all requests
func NewTimeoutMiddleware() *middleware.TimeoutMiddleware {
	return middleware.NewTimeoutMiddleware(middleware.TimeoutConfig{
		Timeout: 30 * time.Second, // 30 seconds for all requests
	})
}

func main() {
	app := InitApp()

	if err := app.Start(context.Background()); err != nil {
		panic(err)
	}

	<-app.Done()
}

func InitApp() *fx.App {
	options := injection.GetOptions()
	options = append(options,
		fx.NopLogger,
		fx.Provide(
			log.NewLogger,
			log.NewGormLogger,
			event.NewSyncEventBus,
			dal.NewGormDB,
			cache.NewCache,
			config.NewConfig,
			handler.NewServer,
			template.NewTemplate,
			middleware.NewAuthMiddleware,
			middleware.NewCSRFMiddleware,
			NewLoginRateLimitMiddleware,
			NewTimeoutMiddleware,
			middleware.NewLocaleMiddleware,
			middleware.NewRequestIDMiddleware,
			middleware.NewLoggerMiddleware,
			middleware.NewRecoveryMiddleware,
			middleware.NewInstallRedirectMiddleware,
		),
		// Removed fx.Populate calls - dependencies should be passed explicitly
		fx.Invoke(
			listener.NewStartListener,
			listener.NewTemplateConfigListener,
			listener.NewLogEventListener,
			listener.NewPostUpdateListener,
			listener.NewCommentListener,
			extension.RegisterCategoryFunc,
			extension.RegisterCommentFunc,
			extension.RegisterTagFunc,
			extension.RegisterMenuFunc,
			extension.RegisterPhotoFunc,
			extension.RegisterLinkFunc,
			extension.RegisterToolFunc,
			extension.RegisterPaginationFunc,
			extension.RegisterPostFunc,
			extension.RegisterStatisticFunc,
			func(s *handler.Server, bus event.Bus) {
				s.RegisterRouters()
				// Publish start event after routes are registered
				bus.Publish(context.Background(), &event.StartEvent{})
			},
		),
	)
	app := fx.New(
		options...,
	)
	return app
}
