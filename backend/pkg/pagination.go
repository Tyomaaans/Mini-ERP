package pkg

import (
	"net/url"
	"strconv"
	"strings"
)

type ListQuery struct {
	Page      int
	PerPage   int
	Sort      string
	Direction string
	Search    string
	Filters   map[string]string
}

func ParsePagination(values url.Values, sorts, filters []string, searchable ...bool) (ListQuery, error) {
	query := ListQuery{
		Page:      1,
		PerPage:   20,
		Sort:      sorts[0],
		Direction: "desc",
		Filters:   make(map[string]string),
	}
	allowed := map[string]bool{"page": true, "per_page": true, "sort": true, "direction": true, "search": true}
	for _, key := range filters {
		allowed[key] = true
	}
	for key, value := range values {
		if (!allowed[key] || (key == "search" && len(searchable) > 0 && !searchable[0])) || len(value) != 1 {
			return query, ErrInvalidInput
		}
	}
	var err error
	if raw := values.Get("page"); raw != "" {
		query.Page, err = strconv.Atoi(raw)
		if err != nil {
			return query, ErrInvalidInput
		}
	}
	if raw := values.Get("per_page"); raw != "" {
		query.PerPage, err = strconv.Atoi(raw)
		if err != nil {
			return query, ErrInvalidInput
		}
	}
	if query.Page < 1 || query.Page > 1000000 || query.PerPage < 1 || query.PerPage > 100 {
		return query, ErrInvalidInput
	}
	if raw := values.Get("sort"); raw != "" {
		found := false
		for _, sort := range sorts {
			if raw == sort {
				found = true
			}
		}
		if !found {
			return query, ErrInvalidInput
		}
		query.Sort = raw
	}
	if raw := values.Get("direction"); raw != "" {
		if raw != "asc" && raw != "desc" {
			return query, ErrInvalidInput
		}
		query.Direction = raw
	}
	query.Search = strings.TrimSpace(values.Get("search"))
	if len(query.Search) > 200 {
		return query, ErrInvalidInput
	}
	for _, key := range filters {
		if raw := values.Get(key); raw != "" {
			if strings.HasSuffix(key, "_id") || key == "created_by" || key == "approved_by" {
				if err := ValidateID(raw); err != nil {
					return query, err
				}
			}
			if key == "is_active" || key == "overdue" {
				if raw != "true" && raw != "false" {
					return query, ErrInvalidInput
				}
			}
			if key == "from" || key == "to" {
				if _, err := ParseDate(raw); err != nil {
					return query, err
				}
			}
			query.Filters[key] = raw
		}
	}
	if query.Filters["from"] != "" && query.Filters["to"] != "" && query.Filters["from"] > query.Filters["to"] {
		return query, ErrInvalidInput
	}
	return query, nil
}

func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.PerPage
}

func (q ListQuery) Order(columns map[string]string) string {
	column, ok := columns[q.Sort]
	if !ok {
		column = columns["created_at"]
	}
	direction := "DESC"
	if q.Direction == "asc" {
		direction = "ASC"
	}
	return column + " " + direction + ", id " + direction
}
