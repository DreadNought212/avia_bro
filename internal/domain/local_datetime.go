package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// LocalDateTime представляет дату и время без информации о часовом поясе.
type LocalDateTime time.Time

// Scan позволяет database/sql сканировать значение PostgreSQL.
func (t *LocalDateTime) Scan(value any) error {
	if value == nil {
		*t = LocalDateTime(time.Time{})
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		*t = LocalDateTime(v)
		return nil

	case string:
		parsed, err := parseLocalDateTime(v)
		if err != nil {
			return err
		}
		*t = LocalDateTime(parsed)
		return nil

	case []byte:
		parsed, err := parseLocalDateTime(string(v))
		if err != nil {
			return err
		}
		*t = LocalDateTime(parsed)
		return nil

	default:
		return fmt.Errorf("cannot scan %T into LocalDateTime", value)
	}
}

func parseLocalDateTime(value string) (time.Time, error) {
	layouts := []string{
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid local datetime: %s", value)
}

// MarshalJSON возвращает время без часового пояса.
func (t LocalDateTime) MarshalJSON() ([]byte, error) {
	value := time.Time(t).Format("2006-01-02T15:04:05.999999999")
	return json.Marshal(value)
}
