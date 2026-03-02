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

	EnableM04SubmissionProjection bool
	EnableM04DeadlineCompletion   bool
	EnableM26AutoApprove          bool
	EnableM26ViewLock             bool
	EnableM08SubmissionConsumer   bool
	EnableM08CampaignConsumer     bool
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

		EnableM04SubmissionProjection: envBool("ENABLE_M04_SUBMISSION_PROJECTION", true),
		EnableM04DeadlineCompletion:   envBool("ENABLE_M04_DEADLINE_COMPLETION", true),
		EnableM26AutoApprove:          envBool("ENABLE_M26_AUTO_APPROVE", true),
		EnableM26ViewLock:             envBool("ENABLE_M26_VIEW_LOCK", true),
		EnableM08SubmissionConsumer:   envBool("ENABLE_M08_SUBMISSION_CONSUMER", true),
		EnableM08CampaignConsumer:     envBool("ENABLE_M08_CAMPAIGN_CONSUMER", true),
	}, nil
}

func envBool(name string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
