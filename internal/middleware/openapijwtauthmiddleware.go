package middleware

import (
	"PowerX/internal/config"
	"PowerX/internal/types"
	"PowerX/internal/types/errorx"
	"PowerX/internal/uc"
	"PowerX/internal/uc/openapi"
	"PowerX/internal/uc/powerx/crm/customerdomain"
	"context"
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
	"strings"
)

type OpenAPIJWTAuthMiddleware struct {
	conf *config.Config
	px   *uc.PowerXUseCase
}

func NewOpenAPIJWTAuthMiddleware(conf *config.Config, px *uc.PowerXUseCase, opts ...OptionFunc) *OpenAPIJWTAuthMiddleware {
	return &OpenAPIJWTAuthMiddleware{
		conf: conf,
		px:   px,
	}
}

func (m *OpenAPIJWTAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	secret := m.conf.OpenAPI.Platforms.BrainX.SecretKey
	unAuth := errorx.ErrUnAuthorization.(*errorx.Error)

	return func(writer http.ResponseWriter, request *http.Request) {

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

		// 获取对接平台的platformId
		payload, err := customerdomain.GetPayloadFromToken(token.Raw)
		if err != nil {
			logx.WithContext(request.Context()).Error(err)
			httpx.Error(writer, errorx.WithCause(unAuth, "无效客户信息"))
			return
		}
		platformId := payload[openapi.AuthPlatformIdKey]
		ctx := context.WithValue(request.Context(), openapi.AuthPlatformKey, platformId)

		// Pass through to next handler if need
		next(writer, request.WithContext(ctx))
	}
}
