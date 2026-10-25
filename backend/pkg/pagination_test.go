package pkg

import (
	"net/url"
	"testing"
)

func TestPaginationRejectsUnboundedAndUnapprovedQueries(t *testing.T) {
	for _, raw := range []string{"page=0", "page=1000001", "per_page=101", "sort=password_hash", "direction=desc%3BDROP+TABLE+users", "debug=true", "page=1&page=2", "warehouse_id=invalid", "from=2026-13-01", "from=2026-10-20&to=2026-10-01", "is_active=1"} {
		t.Run(raw, func(t *testing.T) {
			values, _ := url.ParseQuery(raw)
			if _, err := ParsePagination(values, []string{"created_at", "name"}, []string{"warehouse_id", "from", "to", "is_active"}); err == nil {
				t.Fatal("query was accepted")
			}
		})
	}
	values, _ := url.ParseQuery("page=2&per_page=10&sort=name&direction=asc")
	query, err := ParsePagination(values, []string{"created_at", "name"}, nil)
	if err != nil || query.Offset() != 10 || query.Order(map[string]string{"name": "name", "created_at": "created_at"}) != "name ASC, id ASC" {
		t.Fatalf("query=%+v err=%v", query, err)
	}
}
