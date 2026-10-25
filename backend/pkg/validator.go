package pkg

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func NewValidator() *validator.Validate {
	return validator.New()
}

func BindJSON(c *gin.Context, value any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return NewError("INVALID_JSON", "Body must be a JSON object with allowed fields", 400)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return NewError("INVALID_JSON", "Body must contain one JSON value", 400)
	}
	return nil
}

func BindAction(c *gin.Context) error {
	if c.Request.Body == nil {
		return nil
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var value struct{}
	if err := decoder.Decode(&value); err != nil {
		if err == io.EOF {
			return nil
		}
		return NewError("INVALID_JSON", "This action accepts an empty body or an empty JSON object", 400)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return NewError("INVALID_JSON", "Body must contain one JSON value", 400)
	}
	return nil
}

func ValidateStruct(validate *validator.Validate, value any) error {
	if err := validate.Struct(value); err != nil {
		return ErrInvalidInput
	}
	return nil
}

func ValidateID(id string) error {
	parsed, err := uuid.Parse(id)
	if err != nil || parsed.String() != id {
		return ErrInvalidInput
	}
	return nil
}

var decimalPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)

type NullableString struct {
	Value *string
	Set   bool
}

func (n *NullableString) UnmarshalJSON(data []byte) error {
	n.Set = true
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	n.Value = &value
	return nil
}

func ParseDecimal(raw string, scale int32, positive bool) (decimal.Decimal, error) {
	value, err := decimal.NewFromString(raw)
	if err != nil || !decimalPattern.MatchString(raw) || value.Exponent() < -scale || value.IsNegative() || (positive && value.IsZero()) {
		return decimal.Zero, ErrInvalidInput
	}
	limit := decimal.New(1, 18-scale)
	if value.GreaterThanOrEqual(limit) {
		return decimal.Zero, ErrInvalidInput
	}
	return value, nil
}

func ParseDate(raw string) (time.Time, error) {
	value, err := time.Parse(time.DateOnly, raw)
	if err != nil {
		return time.Time{}, ErrInvalidInput
	}
	return value, nil
}
