package pkg

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestInternalErrorsHideCredentialsAndIncludeRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request = c.Request.WithContext(WithRequestID(c.Request.Context(), "00000000-0000-0000-0000-000000000001"))
	ErrorResponse(c, errors.New("password=private database connection string"))
	body := recorder.Body.String()
	if recorder.Code != 500 || strings.Contains(body, "password") || !strings.Contains(body, "INTERNAL_ERROR") || !strings.Contains(body, "00000000-0000-0000-0000-000000000001") {
		t.Fatalf("unsafe response: %s", body)
	}
}
