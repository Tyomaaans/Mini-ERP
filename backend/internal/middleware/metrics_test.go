package middleware

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

func TestMetricsSnapshotPreservesHistogramCounts(t *testing.T) {
	metrics := &Metrics{}
	var workers sync.WaitGroup
	for i := 0; i < 100; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			metrics.Observe(200, 10*time.Millisecond)
		}()
	}
	workers.Wait()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/api/v1/metrics", nil)
	c.Request = c.Request.WithContext(pkg.WithActor(c.Request.Context(), pkg.Actor{
		ID:          "admin",
		Permissions: []string{"system.metrics.read"},
	}))
	metrics.Handler(c)
	body := recorder.Body.String()
	for _, expected := range []string{"http_requests_total{status_class=\"2xx\"} 100", "http_request_duration_seconds_bucket{le=\"+Inf\"} 100", "http_request_duration_seconds_count 100"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("missing metric %s", expected)
		}
	}
}
