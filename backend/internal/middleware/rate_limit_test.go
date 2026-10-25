package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestRateLimitUsesRemainingRedisWindow(t *testing.T) {
	address := os.Getenv("TEST_REDIS_ADDR")
	if address == "" {
		t.Skip("set TEST_REDIS_ADDR to a dedicated disposable Redis service")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: os.Getenv("TEST_REDIS_PASSWORD"),
	})
	t.Cleanup(func() {
		_ = client.Close()
	})
	name := "test." + uuid.NewString()
	hash := sha256.Sum256([]byte("192.0.2.1"))
	key := "rate:" + name + ":" + hex.EncodeToString(hash[:])
	t.Cleanup(func() {
		client.Del(context.Background(), key)
	})

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	_ = router.SetTrustedProxies(nil)
	router.GET("/", NewRateLimiter(client).Limit(name, 1, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	request := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.1:12345"
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		return response
	}

	if response := request(); response.Code != http.StatusNoContent {
		t.Fatalf("first request: status=%d body=%s", response.Code, response.Body)
	}
	initial := request()
	initialSeconds, err := strconv.Atoi(initial.Header().Get("Retry-After"))
	if err != nil || initial.Code != http.StatusTooManyRequests || initialSeconds < 1 || initialSeconds > 60 {
		t.Fatalf("initial limit response: status=%d retry=%d error=%v", initial.Code, initialSeconds, err)
	}
	if err := client.PExpire(context.Background(), key, 1500*time.Millisecond).Err(); err != nil {
		t.Fatal(err)
	}

	remaining := request()
	seconds, err := strconv.Atoi(remaining.Header().Get("Retry-After"))
	if err != nil || remaining.Code != http.StatusTooManyRequests || seconds < 1 || seconds > 2 || seconds >= initialSeconds {
		t.Fatalf("remaining limit response: status=%d retry=%d initial=%d error=%v", remaining.Code, seconds, initialSeconds, err)
	}
}

func TestRateLimitFailsClosedWhenRedisIsUnavailable(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	_ = client.Close()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.GET("/", NewRateLimiter(client).Limit("unavailable", 1, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable Redis allowed request: status=%d", response.Code)
	}
}
