package oauth2

import (
	"context"
	errs "errors"
	"net/http"

	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
	"github.com/hysios/mx"
	"github.com/hysios/mx/logger"
	"github.com/hysios/mx/middleware"
	"github.com/hysios/mx/provisioning"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

type Option struct {
	Logger                       *zap.Logger
	Server                       *server.Server
	Manager                      *manage.Manager
	TokenStore                   oauth2.TokenStore
	ClientStore                  oauth2.ClientStore
	PassTokenCfg                 *manage.Config
	ClientTokenCfg               *manage.Config
	AllowedResponseType          []oauth2.ResponseType
	ClientFormHandler            func(r *http.Request) (clientID, clientSecret string, err error)
	InternalErrHandler           func(err error) (re *errors.Response)
	ErrHandler                   func(re *errors.Response)
	ErrorHandler                 func(w http.ResponseWriter, err error, statusCode int)
	AuthHandler                  func(w http.ResponseWriter, r *http.Request)
	TokenHandler                 func(w http.ResponseWriter, r *http.Request)
	PasswordAuthorizationHandler func(ctx context.Context, clientID, username, password string) (userID string, err error)
	TokenAuth                    func(ctx context.Context, token oauth2.TokenInfo) context.Context
	ClientAuthHandler            func(clientID string, grant oauth2.GrantType) (allowed bool, err error)
	ClientScopeHandler           func(tgr *oauth2.TokenGenerateRequest) (allowed bool, err error)
	TokenSkipMatchers            []middleware.Matcher
	// AuthError                    func(w http.ResponseWriter, r *http.Request, err error)
}

var Logger *zap.Logger = zap.L()

var DefaultOption = Option{
	Logger:             zap.L(),
	Manager:            manage.NewDefaultManager(),
	ClientStore:        store.NewClientStore(),
	ClientFormHandler:  server.ClientFormHandler,
	InternalErrHandler: InternalErrHandler,
	ErrHandler:         ErrHandler,
	ErrorHandler: func(w http.ResponseWriter, err error, statusCode int) {
		http.Error(w, err.Error(), statusCode)
	},
	AllowedResponseType: []oauth2.ResponseType{
		oauth2.Code, oauth2.Token,
	},
	AuthHandler:  func(w http.ResponseWriter, r *http.Request) {},
	TokenHandler: func(w http.ResponseWriter, r *http.Request) {},
	TokenAuth: func(ctx context.Context, token oauth2.TokenInfo) context.Context {
		return ctx
	},
	// AuthError: AuthError,
}

func InternalErrHandler(err error) (re *errors.Response) {
	Logger.Warn("Internal Error:", zap.Error(err))
	return
}

func ErrHandler(re *errors.Response) {
	Logger.Warn("Error:", zap.Any("response", re))
}

func (o *Option) AuthError(w http.ResponseWriter, r *http.Request, err error) {
	Logger.Debug("Auth Error:", zap.Error(err))
	if errs.Is(err, errors.ErrInvalidAccessToken) {
		o.ErrorHandler(w, err, http.StatusUnauthorized)
	} else if errs.Is(err, errors.ErrExpiredAccessToken) {
		o.ErrorHandler(w, err, http.StatusForbidden)
	} else {
		o.ErrorHandler(w, err, http.StatusBadRequest)
	}
}

func Middleware(opt Option) mx.Middleware {
	if opt.Logger != nil {
		Logger = opt.Logger
	}

	if opt.TokenStore == nil {
		opt.TokenStore, _ = store.NewMemoryTokenStore()
	}

	if opt.Manager == nil {
		opt.Manager = DefaultOption.Manager
	}

	if opt.PassTokenCfg != nil {
		opt.Manager.SetPasswordTokenCfg(opt.PassTokenCfg)
	}

	if opt.ClientTokenCfg != nil {
		opt.Manager.SetClientTokenCfg(opt.ClientTokenCfg)
	}

	opt.Manager.MapTokenStorage(opt.TokenStore)

	if opt.ClientStore == nil {
		opt.ClientStore = DefaultOption.ClientStore
	}

	opt.Manager.MapClientStorage(opt.ClientStore)

	if opt.InternalErrHandler == nil {
		opt.InternalErrHandler = DefaultOption.InternalErrHandler
	}

	if opt.ErrHandler == nil {
		opt.ErrHandler = DefaultOption.ErrHandler
	}

	if opt.ErrorHandler == nil {
		opt.ErrorHandler = DefaultOption.ErrorHandler
	}

	if opt.AuthHandler == nil {
		opt.AuthHandler = DefaultOption.AuthHandler
	}

	if opt.TokenHandler == nil {
		opt.TokenHandler = DefaultOption.TokenHandler
	}

	if opt.TokenAuth == nil {
		opt.TokenAuth = DefaultOption.TokenAuth
	}

	// if opt.AuthError == nil {
	// 	// opt.AuthError = DefaultOption.AuthError
	// }

	if opt.Server == nil {
		opt.Server = server.NewDefaultServer(opt.Manager)
	}

	opt.Server.SetInternalErrorHandler(opt.InternalErrHandler)
	opt.Server.SetResponseErrorHandler(opt.ErrHandler)
	opt.Server.SetAllowGetAccessRequest(true)

	if opt.ClientFormHandler == nil {
		opt.ClientFormHandler = DefaultOption.ClientFormHandler
	}

	opt.Server.SetClientInfoHandler(opt.ClientFormHandler)
	opt.Server.SetAllowedResponseType(opt.AllowedResponseType...)
	if opt.ClientAuthHandler == nil {
		opt.Server.SetClientAuthorizedHandler(opt.ClientAuthHandler)
	}

	if opt.ClientScopeHandler != nil {
		opt.Server.SetClientScopeHandler(opt.ClientScopeHandler)
	}

	if opt.PasswordAuthorizationHandler != nil {
		opt.Server.SetPasswordAuthorizationHandler(opt.PasswordAuthorizationHandler)
	}

	provisioning.Provision(func(gw *mx.Gateway) {
		gw.HandlePrefix("/oauth2", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger.Logger.Info("oauth2 middleware", zap.String("path", r.URL.Path))
			switch {
			case r.URL.Path == "/oauth2/authorize":
				err := opt.Server.HandleAuthorizeRequest(w, r)
				if err != nil {
					opt.ErrorHandler(w, err, http.StatusBadRequest)
					// http.Error(w, err.Error(), http.StatusBadRequest)
				}
			case r.URL.Path == "/oauth2/token":
				var ctx = r.Context()
				ctx = metadata.AppendToOutgoingContext(ctx,
					"x-forwarded-for", RemoteAddr(r),
					"grpcgateway-user-agent", r.UserAgent(),
				)
				opt.Server.HandleTokenRequest(w, r.WithContext(ctx))
			default:
				var (
					ctx   = r.Context()
					token oauth2.TokenInfo
					err   error
				)

				// 跳过 token 验证, 根据配置的 TokenSkipMatchers 判断是否跳过
				for _, matcher := range opt.TokenSkipMatchers {
					if matcher(r) {
						next.ServeHTTP(w, r)
						return
					}
				}

				token, err = opt.Server.ValidationBearerToken(r)
				if err != nil {
					opt.AuthError(w, r, err)
					return
				}

				ctx = opt.TokenAuth(ctx, token)
				// 给予 token 关联到 ctx 的机会，以便后续使用提供给其他中间件做鉴权
				next.ServeHTTP(w, r.WithContext(ctx))
			}

			return
		})
	}
}

func RemoteAddr(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
