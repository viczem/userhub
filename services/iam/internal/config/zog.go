package config

import (
	"fmt"
	"time"

	z "github.com/Oudwins/zog"
)

func parseDuration(value any) (any, error) {
	switch value := value.(type) {
	case time.Duration:
		return value, nil
	case string:
		duration, err := time.ParseDuration(value)
		if err != nil {
			return nil, fmt.Errorf("invalid duration %q: %w", value, err)
		}

		return duration, nil
	default:
		return nil, fmt.Errorf("duration must be a string, got %T", value)
	}
}

func durationSchema(defaultValue time.Duration) *z.NumberSchema[time.Duration] {
	return z.IntLike[time.Duration](
		z.WithCoercer(parseDuration),
	).Default(defaultValue).GTE(1*time.Second, z.Message("must be at least 1s"))
}

func nonNegativeDurationSchema(defaultValue time.Duration) *z.NumberSchema[time.Duration] {
	return z.IntLike[time.Duration](
		z.WithCoercer(parseDuration),
	).Default(defaultValue).GTE(0)
}
