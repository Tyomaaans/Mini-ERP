package middleware

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type Metrics struct {
	mu           sync.RWMutex
	requests     [6]uint64
	count        uint64
	microseconds uint64
	buckets      [11]uint64
}

var durationBounds = [...]time.Duration{5 * time.Millisecond, 10 * time.Millisecond, 25 * time.Millisecond, 50 * time.Millisecond, 100 * time.Millisecond, 250 * time.Millisecond, 500 * time.Millisecond, time.Second, 2500 * time.Millisecond, 5 * time.Second, 10 * time.Second}

func (m *Metrics) Observe(status int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	class := status / 100
	if class < 1 || class > 5 {
		class = 5
	}
	m.requests[class]++
	m.count++
	m.microseconds += uint64(duration.Microseconds())
	for i, bound := range durationBounds {
		if duration <= bound {
			m.buckets[i]++
		}
	}
}

func (m *Metrics) Handler(c *gin.Context) {
	if err := pkg.ActorFrom(c.Request.Context()).Require("system.metrics.read"); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}
	m.mu.RLock()
	requests, count, microseconds, buckets := m.requests, m.count, m.microseconds, m.buckets
	m.mu.RUnlock()
	var output strings.Builder
	output.WriteString("# HELP http_requests_total HTTP requests by status class.\n# TYPE http_requests_total counter\n")
	for class := 1; class <= 5; class++ {
		fmt.Fprintf(&output, "http_requests_total{status_class=\"%dxx\"} %d\n", class, requests[class])
	}
	output.WriteString("# HELP http_request_duration_seconds HTTP request duration.\n# TYPE http_request_duration_seconds histogram\n")
	for i, bound := range durationBounds {
		fmt.Fprintf(&output, "http_request_duration_seconds_bucket{le=\"%g\"} %d\n", bound.Seconds(), buckets[i])
	}
	fmt.Fprintf(&output, "http_request_duration_seconds_bucket{le=\"+Inf\"} %d\nhttp_request_duration_seconds_sum %g\nhttp_request_duration_seconds_count %d\n", count, float64(microseconds)/1e6, count)
	c.Data(200, "text/plain; version=0.0.4; charset=utf-8", []byte(output.String()))
}
