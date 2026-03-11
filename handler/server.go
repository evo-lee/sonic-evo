package handler

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	hertzserver "github.com/cloudwego/hertz/pkg/app/server"
	"go.uber.org/dig"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/go-sonic/sonic/config"
	"github.com/go-sonic/sonic/event"
	"github.com/go-sonic/sonic/handler/admin"
	"github.com/go-sonic/sonic/handler/content"
	"github.com/go-sonic/sonic/handler/content/api"
	"github.com/go-sonic/sonic/handler/middleware"
	"github.com/go-sonic/sonic/handler/web"
	"github.com/go-sonic/sonic/handler/web/hertzadapter"
	"github.com/go-sonic/sonic/model/dto"
	"github.com/go-sonic/sonic/service"
	"github.com/go-sonic/sonic/template"
	"github.com/go-sonic/sonic/util/xerr"
)

type Server struct {
	logger                    *zap.Logger
	Config                    *config.Config
	HertzServer               *hertzserver.Hertz
	Router                    web.Router
	Template                  *template.Template
	AuthMiddleware            *middleware.AuthMiddleware
	CSRFMiddleware            *middleware.CSRFMiddleware
	LoginRateLimitMiddleware  *middleware.RateLimitMiddleware
	TimeoutMiddleware         *middleware.TimeoutMiddleware
	LocaleMiddleware          *middleware.LocaleMiddleware
	RequestIDMiddleware       *middleware.RequestIDMiddleware
	LogMiddleware             *middleware.LoggerMiddleware
	RecoveryMiddleware        *middleware.RecoveryMiddleware
	InstallRedirectMiddleware *middleware.InstallRedirectMiddleware
	OptionService             service.OptionService
	ThemeService              service.ThemeService
	SheetService              service.SheetService
	AdminHandler              *admin.AdminHandler
	AttachmentHandler         *admin.AttachmentHandler
	BackupHandler             *admin.BackupHandler
	CategoryHandler           *admin.CategoryHandler
	InstallHandler            *admin.InstallHandler
	JournalHandler            *admin.JournalHandler
	JournalCommentHandler     *admin.JournalCommentHandler
	LinkHandler               *admin.LinkHandler
	LogHandler                *admin.LogHandler
	MenuHandler               *admin.MenuHandler
	OptionHandler             *admin.OptionHandler
	PhotoHandler              *admin.PhotoHandler
	PostHandler               *admin.PostHandler
	PostCommentHandler        *admin.PostCommentHandler
	SheetHandler              *admin.SheetHandler
	SheetCommentHandler       *admin.SheetCommentHandler
	StatisticHandler          *admin.StatisticHandler
	TagHandler                *admin.TagHandler
	ThemeHandler              *admin.ThemeHandler
	UserHandler               *admin.UserHandler
	EmailHandler              *admin.EmailHandler
	IndexHandler              *content.IndexHandler
	FeedHandler               *content.FeedHandler
	ArchiveHandler            *content.ArchiveHandler
	ViewHandler               *content.ViewHandler
	ContentCategoryHandler    *content.CategoryHandler
	ContentSheetHandler       *content.SheetHandler
	ContentTagHandler         *content.TagHandler
	ContentLinkHandler        *content.LinkHandler
	ContentPhotoHandler       *content.PhotoHandler
	ContentJournalHandler     *content.JournalHandler
	ContentSearchHandler      *content.SearchHandler
	ContentAPIArchiveHandler  *api.ArchiveHandler
	ContentAPICategoryHandler *api.CategoryHandler
	ContentAPIJournalHandler  *api.JournalHandler
	ContentAPILinkHandler     *api.LinkHandler
	ContentAPIPostHandler     *api.PostHandler
	ContentAPISheetHandler    *api.SheetHandler
	ContentAPIOptionHandler   *api.OptionHandler
	ContentAPIPhotoHandler    *api.PhotoHandler
	ContentAPICommentHandler  *api.CommentHandler
}

type ServerParams struct {
	dig.In
	Config                    *config.Config
	Logger                    *zap.Logger
	Event                     event.Bus
	Template                  *template.Template
	AuthMiddleware            *middleware.AuthMiddleware
	CSRFMiddleware            *middleware.CSRFMiddleware
	LoginRateLimitMiddleware  *middleware.RateLimitMiddleware
	TimeoutMiddleware         *middleware.TimeoutMiddleware
	LocaleMiddleware          *middleware.LocaleMiddleware
	RequestIDMiddleware       *middleware.RequestIDMiddleware
	LogMiddleware             *middleware.LoggerMiddleware
	RecoveryMiddleware        *middleware.RecoveryMiddleware
	InstallRedirectMiddleware *middleware.InstallRedirectMiddleware
	OptionService             service.OptionService
	ThemeService              service.ThemeService
	SheetService              service.SheetService
	AdminHandler              *admin.AdminHandler
	AttachmentHandler         *admin.AttachmentHandler
	BackupHandler             *admin.BackupHandler
	CategoryHandler           *admin.CategoryHandler
	InstallHandler            *admin.InstallHandler
	JournalHandler            *admin.JournalHandler
	JournalCommentHandler     *admin.JournalCommentHandler
	LinkHandler               *admin.LinkHandler
	LogHandler                *admin.LogHandler
	MenuHandler               *admin.MenuHandler
	OptionHandler             *admin.OptionHandler
	PhotoHandler              *admin.PhotoHandler
	PostHandler               *admin.PostHandler
	PostCommentHandler        *admin.PostCommentHandler
	SheetHandler              *admin.SheetHandler
	SheetCommentHandler       *admin.SheetCommentHandler
	StatisticHandler          *admin.StatisticHandler
	TagHandler                *admin.TagHandler
	ThemeHandler              *admin.ThemeHandler
	UserHandler               *admin.UserHandler
	EmailHandler              *admin.EmailHandler
	IndexHandler              *content.IndexHandler
	FeedHandler               *content.FeedHandler
	ArchiveHandler            *content.ArchiveHandler
	ViewHandler               *content.ViewHandler
	ContentCategoryHandler    *content.CategoryHandler
	ContentSheetHandler       *content.SheetHandler
	ContentTagHandler         *content.TagHandler
	ContentLinkHandler        *content.LinkHandler
	ContentPhotoHandler       *content.PhotoHandler
	ContentJournalHandler     *content.JournalHandler
	ContentSearchHandler      *content.SearchHandler
	ContentAPIArchiveHandler  *api.ArchiveHandler
	ContentAPICategoryHandler *api.CategoryHandler
	ContentAPIJournalHandler  *api.JournalHandler
	ContentAPILinkHandler     *api.LinkHandler
	ContentAPIPostHandler     *api.PostHandler
	ContentAPISheetHandler    *api.SheetHandler
	ContentAPIOptionHandler   *api.OptionHandler
	ContentAPIPhotoHandler    *api.PhotoHandler
	ContentAPICommentHandler  *api.CommentHandler
}

func NewServer(param ServerParams, lifecycle fx.Lifecycle) *Server {
	conf := param.Config
	hertzEngine := hertzserver.New(
		hertzserver.WithHostPorts(fmt.Sprintf("%s:%s", conf.Server.Host, conf.Server.Port)),
	)
	router := hertzadapter.NewRouter(hertzEngine)

	s := &Server{
		logger:                    param.Logger,
		Config:                    param.Config,
		HertzServer:               hertzEngine,
		Router:                    router,
		Template:                  param.Template,
		AuthMiddleware:            param.AuthMiddleware,
		CSRFMiddleware:            param.CSRFMiddleware,
		LoginRateLimitMiddleware:  param.LoginRateLimitMiddleware,
		TimeoutMiddleware:         param.TimeoutMiddleware,
		LocaleMiddleware:          param.LocaleMiddleware,
		RequestIDMiddleware:       param.RequestIDMiddleware,
		LogMiddleware:             param.LogMiddleware,
		RecoveryMiddleware:        param.RecoveryMiddleware,
		InstallRedirectMiddleware: param.InstallRedirectMiddleware,
		AdminHandler:              param.AdminHandler,
		AttachmentHandler:         param.AttachmentHandler,
		BackupHandler:             param.BackupHandler,
		CategoryHandler:           param.CategoryHandler,
		InstallHandler:            param.InstallHandler,
		JournalHandler:            param.JournalHandler,
		JournalCommentHandler:     param.JournalCommentHandler,
		LinkHandler:               param.LinkHandler,
		LogHandler:                param.LogHandler,
		MenuHandler:               param.MenuHandler,
		OptionHandler:             param.OptionHandler,
		PhotoHandler:              param.PhotoHandler,
		PostHandler:               param.PostHandler,
		PostCommentHandler:        param.PostCommentHandler,
		SheetHandler:              param.SheetHandler,
		SheetCommentHandler:       param.SheetCommentHandler,
		StatisticHandler:          param.StatisticHandler,
		TagHandler:                param.TagHandler,
		ThemeHandler:              param.ThemeHandler,
		UserHandler:               param.UserHandler,
		EmailHandler:              param.EmailHandler,
		OptionService:             param.OptionService,
		ThemeService:              param.ThemeService,
		SheetService:              param.SheetService,
		IndexHandler:              param.IndexHandler,
		FeedHandler:               param.FeedHandler,
		ArchiveHandler:            param.ArchiveHandler,
		ViewHandler:               param.ViewHandler,
		ContentCategoryHandler:    param.ContentCategoryHandler,
		ContentSheetHandler:       param.ContentSheetHandler,
		ContentTagHandler:         param.ContentTagHandler,
		ContentLinkHandler:        param.ContentLinkHandler,
		ContentPhotoHandler:       param.ContentPhotoHandler,
		ContentJournalHandler:     param.ContentJournalHandler,
		ContentAPIArchiveHandler:  param.ContentAPIArchiveHandler,
		ContentAPICategoryHandler: param.ContentAPICategoryHandler,
		ContentAPIJournalHandler:  param.ContentAPIJournalHandler,
		ContentAPILinkHandler:     param.ContentAPILinkHandler,
		ContentAPIPostHandler:     param.ContentAPIPostHandler,
		ContentAPISheetHandler:    param.ContentAPISheetHandler,
		ContentAPIOptionHandler:   param.ContentAPIOptionHandler,
		ContentSearchHandler:      param.ContentSearchHandler,
		ContentAPIPhotoHandler:    param.ContentAPIPhotoHandler,
		ContentAPICommentHandler:  param.ContentAPICommentHandler,
	}
	lifecycle.Append(fx.Hook{OnStart: s.Run, OnStop: s.Stop})
	return s
}

func (s *Server) Run(ctx context.Context) error {
	go func() {
		if err := s.HertzServer.Run(); err != nil {
			s.logger.Error("unexpected error from hertz Run", zap.Error(err))
			fmt.Printf("http server start error:%s\n", err.Error())
			os.Exit(1)
		}
	}()
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.HertzServer != nil {
		return s.HertzServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) wrapHandler(handler any) web.HandlerFunc {
	return func(ctx web.Context) {
		var (
			data any
			err  error
		)
		switch h := handler.(type) {
		case func(web.Context) (interface{}, error):
			data, err = h(ctx)
		default:
			panic("unsupported handler type")
		}
		if err != nil {
			status := xerr.GetHTTPStatus(err)
			code := middleware.ErrorCodeFromError(err)
			message := xerr.GetMessage(err)
			if message == "" || message == http.StatusText(status) {
				message = middleware.LocalizedHTTPStatusText(ctx, status)
			}
			requestID := middleware.GetRequestID(ctx)
			s.logger.Error("handler error",
				zap.Error(err),
				zap.Int("status", status),
				zap.String("code", code),
				zap.String("request_id", requestID),
				zap.String("method", ctx.Method()),
				zap.String("path", ctx.Path()),
			)
			ctx.JSON(status, middleware.BuildErrorDTO(ctx, status, code, message))
			return
		}

		ctx.JSON(http.StatusOK, &dto.BaseDTO{
			Status:  http.StatusOK,
			Data:    data,
			Message: middleware.T(ctx, "common.ok", "OK"),
		})
	}
}

type wrapperHTMLHandler func(ctx web.Context, model template.Model) (templateName string, err error)

var (
	htmlContentType = []string{"text/html; charset=utf-8"}
	xmlContentType  = []string{"application/xml; charset=utf-8"}
)

func (s *Server) wrapHTMLHandler(handler wrapperHTMLHandler) web.HandlerFunc {
	return func(ctx web.Context) {
		model := template.Model{}
		templateName, err := handler(ctx, model)
		if err != nil {
			s.handleError(ctx, err)
			return
		}
		if templateName == "" {
			return
		}
		if ctx.ResponseHeader("Content-Type") == "" {
			ctx.SetHeader("Content-Type", htmlContentType[0])
		}
		err = s.Template.ExecuteTemplate(ctx.Writer(), templateName, model)
		if err != nil {
			s.logger.Error("render template err", zap.Error(err))
		}
	}
}

func (s *Server) wrapTextHandler(handler wrapperHTMLHandler) web.HandlerFunc {
	return func(ctx web.Context) {
		model := template.Model{}
		templateName, err := handler(ctx, model)
		if err != nil {
			s.handleError(ctx, err)
			return
		}
		if ctx.ResponseHeader("Content-Type") == "" {
			ctx.SetHeader("Content-Type", xmlContentType[0])
		}
		err = s.Template.ExecuteTextTemplate(ctx.Writer(), templateName, model)
		if err != nil {
			s.logger.Error("render template err", zap.Error(err))
		}
	}
}

func (s *Server) handleError(ctx web.Context, err error) {
	status := xerr.GetHTTPStatus(err)
	message := xerr.GetMessage(err)
	if message == "" || message == http.StatusText(status) {
		message = middleware.LocalizedHTTPStatusText(ctx, status)
	}
	s.logger.Error("render html/text handler error",
		zap.Error(err),
		zap.Int("status", status),
		zap.String("request_id", middleware.GetRequestID(ctx)),
		zap.String("method", ctx.Method()),
		zap.String("path", ctx.Path()),
	)
	model := template.Model{}

	templateName, _ := s.ThemeService.Render(ctx.RequestContext(), strconv.Itoa(status))
	t := s.Template.HTMLTemplate.Lookup(templateName)
	if t == nil {
		templateName = "common/error/error"
	}

	if ctx.ResponseHeader("Content-Type") == "" {
		ctx.SetHeader("Content-Type", htmlContentType[0])
	}
	ctx.Status(status)

	model["status"] = status
	model["message"] = message
	model["err"] = err
	model["request_id"] = middleware.GetRequestID(ctx)
	model["error_title"] = middleware.T(ctx, "common.unknown_error", "Unknown Error")
	model["error_default_message"] = middleware.T(ctx, "error.default_message", "Unknown error")
	model["back_home_label"] = middleware.T(ctx, "common.home", "Home")

	err = s.Template.ExecuteTemplate(ctx.Writer(), templateName, model)
	if err != nil {
		s.logger.Error("render error template err", zap.Error(err))
	}
}
