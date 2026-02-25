package config

import (
	"os"
	"strings"
)

// Config is centralized process configuration.
// Keep infra values here and pass typed config into builders.
type Config struct {
	ServiceName  string
	HTTPPort     string
	PostgresDSN  string
	KafkaBrokers []string
}

func Load() (Config, error) {
	service := os.Getenv("SERVICE_NAME")
	if service == "" {
		service = "solomon"
	}

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}

	var brokers []string
	for _, value := range strings.Split(os.Getenv("KAFKA_BROKERS"), ",") {
		value = strings.TrimSpace(value)
		if value != "" {
			brokers = append(brokers, value)
		}
	}
	if len(brokers) == 0 {
		brokers = []string{"localhost:9092"}
	}

	return Config{
		ServiceName:  service,
		HTTPPort:     port,
		PostgresDSN:  os.Getenv("POSTGRES_DSN"),
		KafkaBrokers: brokers,
	}, nil
}
