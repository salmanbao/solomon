package config

// Config is centralized process configuration.
// Keep infra values here and pass typed config into builders.
type Config struct {
	ServiceName string
	HTTPPort    string
	PostgresDSN string
	KafkaBrokers []string
}

func Load() (Config, error) {
	// TODO: load from env and validate required fields.
	return Config{ServiceName: "solomon", HTTPPort: "8080"}, nil
}
