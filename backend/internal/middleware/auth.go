package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"mini-erp/internal/auth-session/user"
	"mini-erp/pkg"
)

type AuthMiddleware struct{ userSvc user.UserService }

func NewAuthMiddleware(userSvc user.UserService) *AuthMiddleware {
	return &AuthMiddleware{userSvc: userSvc}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || parts[0] != "Bearer" {
			pkg.ErrorResponse(c, pkg.ErrUnauthorized)
			return
		}
		actor, sid, err := m.userSvc.Authenticate(c.Request.Context(), parts[1])
		if err != nil {
			pkg.ErrorResponse(c, err)
			return
		}
		c.Request = c.Request.WithContext(pkg.WithActor(c.Request.Context(), actor))
		c.Set("sessionID", sid)
		c.Next()
	}
}
