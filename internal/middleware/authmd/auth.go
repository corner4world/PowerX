package authmd

import (
	"PowerX/internal/svc"
	"PowerX/internal/types"
	"PowerX/internal/types/errorx"
	"PowerX/internal/uc"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"regexp"
	"strings"
)

type option struct {
	public      []string
	disableAuth bool
}

type optionFunc func(opt *option)

func WithPublicPrefix(path ...string) optionFunc {
	return func(opt *option) {
		opt.public = path
	}
}
func DisableToken(b bool) func(opt *option) {
	return func(opt *option) {
		opt.disableAuth = b
	}
}

func AuthMiddleware(ctx *svc.ServiceContext, opts ...optionFunc) rest.Middleware {
	opt := option{}
	for _, o := range opts {
		o(&opt)
	}
	secret := ctx.Config.JWTSecret
	unAuth := errorx.ErrUnAuthorization.(*errorx.Error)
	unKnow := errorx.ErrUnKnow.(*errorx.Error)

	publicRouter := mux.NewRouter()
	for _, s := range opt.public {
		publicRouter.NewRoute().PathPrefix(s)
	}

	resPrefixReg, _ := regexp.Compile(`^/api/\w+`)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			// public
			var match mux.RouteMatch
			if publicRouter.Match(request, &match) {
				next(writer, request)
				return
			}

			// 校验Token
			if opt.disableAuth {
				next(writer, request)
				return
			}

			authorization := request.Header.Get("Authorization")
			splits := strings.Split(authorization, "Bearer")
			if len(splits) != 2 {
				httpx.Error(writer, unAuth)
				return
			}
			tokenString := strings.TrimSpace(splits[1])

			var claims types.TokenClaims
			token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				if errors.Is(err, jwt.ErrTokenMalformed) {
					httpx.Error(writer, unAuth)
				} else if errors.Is(err, jwt.ErrTokenExpired) || errors.Is(err, jwt.ErrTokenNotValidYet) {
					httpx.Error(writer, unAuth)
				} else {
					logx.WithContext(request.Context()).Error(err)
					httpx.Error(writer, errorx.WithCause(unAuth, "违规Token"))
				}
				return
			}

			// temp method map to act
			var op string
			switch request.Method {
			case http.MethodGet:
				op = "read"
				break
			case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
				op = "write"
				break
			}

			// 权限验证
			prefix := strings.TrimPrefix(resPrefixReg.FindString(request.URL.Path), "/api/")
			ok, err := ctx.UC.Auth.Casbin.Enforce(claims.Subject, prefix, op)
			if err != nil {
				httpx.Error(writer, unKnow)
				return
			}
			if !ok {
				httpx.Error(writer, errorx.WithCause(unAuth, "权限不足"))
				return
			}
			request = request.WithContext(ctx.UC.MetadataCtx.WithAuthMetadataCtxValue(request.Context(), &uc.AuthMetadata{
				UID: claims.UID,
			}))
			next(writer, request)
		}
	}
}