package postgresadapter

import "time"

// SystemClock is the default runtime clock implementation.
type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now().UTC()
}
