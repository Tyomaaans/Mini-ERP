package middleware

import (
	"context"
	"log/slog"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"mini-erp/pkg"
)

func Request(logger *slog.Logger, metrics ...*Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		requestID := c.GetHeader("X-Request-ID")
		if pkg.ValidateID(requestID) != nil {
			requestID = uuid.NewString()
		}
		c.Header("X-Request-ID", requestID)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Cache-Control", "no-store")
		c.Request = c.Request.WithContext(pkg.WithRequestID(c.Request.Context(), requestID))
		defer func() {
			if recover() != nil {
				pkg.ErrorResponse(c, pkg.ErrInternal)
			}
			if len(metrics) > 0 {
				metrics[0].Observe(c.Writer.Status(), time.Since(start))
			}
			logger.Info("http_request", "request_id", requestID, "method", c.Request.Method, "route", c.FullPath(), "status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds())
		}()
		if _, err := url.ParseQuery(c.Request.URL.RawQuery); err != nil {
			pkg.ErrorResponse(c, pkg.NewError("INVALID_QUERY", "Query string is malformed", 400))
			return
		}
		c.Next()
	}
}
