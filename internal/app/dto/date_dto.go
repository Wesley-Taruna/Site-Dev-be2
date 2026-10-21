package dto

import (
	"database/sql/driver"
	"strings"
	"time"
)

const dateOnlyLayout = "2006-01-02"

// DateOnly reads and writes JSON as "YYYY-MM-DD" (zero = ""), same as
// site-dev-api's dto.DateOnly.
type DateOnly struct {
	time.Time
}

func (d *DateOnly) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		return nil
	}
	t, err := time.Parse(dateOnlyLayout, s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}

func (d DateOnly) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte(`""`), nil
	}
	return []byte(`"` + d.Time.Format(dateOnlyLayout) + `"`), nil
}

// Value lets DateOnly go straight into a gorm map update.
func (d DateOnly) Value() (driver.Value, error) {
	if d.Time.IsZero() {
		return nil, nil
	}
	return d.Time, nil
}
