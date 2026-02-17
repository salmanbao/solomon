package db

// Postgres wraps DB connectivity.
// Keep transaction helpers here to support outbox + state consistency.
type Postgres struct {}

func Connect(dsn string) (*Postgres, error) {
	// TODO: open connection pool and ping.
	return &Postgres{}, nil
}
