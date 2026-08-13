package timezone

import "time"

const eventDateLayout = "January 2, 2006 at 3:04 PM MST"

// FormatEventDate converts a canonical UTC event timestamp into the event's
// configured IANA timezone for presentation. Invalid or missing zones fall
// back to UTC so notification delivery is never blocked by display metadata.
func FormatEventDate(value time.Time, zone string) string {
	location := locationFor(zone)
	return value.In(location).Format(eventDateLayout)
}

// FormatEventDay is the short, timezone-aware date used in subjects.
func FormatEventDay(value time.Time, zone string) string {
	return value.In(locationFor(zone)).Format("Jan 2")
}

func locationFor(zone string) *time.Location {
	if zone != "" {
		if loaded, err := time.LoadLocation(zone); err == nil {
			return loaded
		}
	}
	return time.UTC
}
