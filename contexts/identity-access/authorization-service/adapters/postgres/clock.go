package postgresadapter

import "time"

// SystemClock implements ports.Clock using wall-clock UTC time.
type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}
