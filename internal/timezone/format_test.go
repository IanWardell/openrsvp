package timezone

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatEventDateUsesEventTimezone(t *testing.T) {
	value := time.Date(2026, time.October, 17, 22, 30, 0, 0, time.UTC)
	assert.Equal(t, "October 17, 2026 at 6:30 PM EDT", FormatEventDate(value, "America/New_York"))
}

func TestFormatEventDateFallsBackToUTC(t *testing.T) {
	value := time.Date(2026, time.October, 17, 22, 30, 0, 0, time.UTC)
	assert.Equal(t, "October 17, 2026 at 10:30 PM UTC", FormatEventDate(value, "Not/AZone"))
}
