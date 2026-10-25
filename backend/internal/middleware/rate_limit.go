package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"mini-erp/pkg"
)

type RateLimiter struct{ client *redis.Client }

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

func (r *RateLimiter) Limit(name string, capacity int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		identity := c.ClientIP()
		if actor := pkg.ActorFrom(c.Request.Context()); actor.ID != "" {
			identity = actor.ID
		}
		hash := sha256.Sum256([]byte(identity))
		key := "rate:" + name + ":" + hex.EncodeToString(hash[:])
		script := redis.NewScript(`
			local n = redis.call('INCR', KEYS[1])
			if n == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
			return {n, redis.call('PTTL', KEYS[1])}`)
		result, err := script.Run(c.Request.Context(), r.client, []string{key}, window.Milliseconds()).Int64Slice()
		if err != nil || len(result) != 2 {
			pkg.ErrorResponse(c, pkg.NewError("SERVICE_UNAVAILABLE", "Rate limiting is temporarily unavailable", 503))
			return
		}
		if result[0] > capacity {
			retryAfter := max(int64(1), (result[1]+999)/1000)
			c.Header("Retry-After", fmt.Sprint(retryAfter))
			pkg.ErrorResponse(c, pkg.NewError("RATE_LIMITED", "Too many requests", 429))
			return
		}
		c.Next()
	}
}
