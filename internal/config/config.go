package config

import (
	"flag"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	envListenAddress = "KFP_EXPORTER_LISTEN_ADDRESS"
	envKFPAPIURL     = "KFP_EXPORTER_KFP_API_URL"
	envNamespaces    = "KFP_EXPORTER_NAMESPACES"
	envPollInterval  = "KFP_EXPORTER_POLL_INTERVAL"
	envPageSize      = "KFP_EXPORTER_PAGE_SIZE"
	envRequestTO     = "KFP_EXPORTER_REQUEST_TIMEOUT"
	envTokenFile     = "KFP_EXPORTER_BEARER_TOKEN_FILE"
)

type Config struct {
	ListenAddress   string
	MetricsPath     string
	HealthPath      string
	KFPAPIURL       string
	Namespaces      []string
	PollInterval    time.Duration
	PageSize        int
	RequestTimeout  time.Duration
	BearerTokenFile string
	ShutdownTimeout time.Duration
}

func FromArgs(args []string, lookupEnv func(string) (string, bool)) (Config, error) {
	pollInterval, err := envDuration(lookupEnv, envPollInterval, time.Minute)
	if err != nil {
		return Config{}, err
	}
	requestTimeout, err := envDuration(lookupEnv, envRequestTO, 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	pageSize, err := envInt(lookupEnv, envPageSize, 100)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		ListenAddress:   envString(lookupEnv, envListenAddress, ":8080"),
		MetricsPath:     "/metrics",
		HealthPath:      "/healthz",
		KFPAPIURL:       envString(lookupEnv, envKFPAPIURL, ""),
		Namespaces:      splitCSV(envString(lookupEnv, envNamespaces, "")),
		PollInterval:    pollInterval,
		PageSize:        pageSize,
		RequestTimeout:  requestTimeout,
		BearerTokenFile: envString(lookupEnv, envTokenFile, ""),
		ShutdownTimeout: 10 * time.Second,
	}

	fs := flag.NewFlagSet("kubeflow-pipelines-exporter", flag.ContinueOnError)
	fs.StringVar(&cfg.ListenAddress, "listen-address", cfg.ListenAddress, "address for the HTTP server")
	fs.StringVar(&cfg.KFPAPIURL, "kfp-api-url", cfg.KFPAPIURL, "Kubeflow Pipelines API base URL")
	namespaces := fs.String("namespaces", strings.Join(cfg.Namespaces, ","), "comma-separated KFP namespaces to collect; empty means all KFP-visible namespaces")
	fs.DurationVar(&cfg.PollInterval, "poll-interval", cfg.PollInterval, "interval between KFP API polls")
	fs.IntVar(&cfg.PageSize, "page-size", cfg.PageSize, "KFP API page size")
	fs.DurationVar(&cfg.RequestTimeout, "request-timeout", cfg.RequestTimeout, "timeout for each KFP API request")
	fs.StringVar(&cfg.BearerTokenFile, "bearer-token-file", cfg.BearerTokenFile, "file containing bearer token for KFP API requests")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg.Namespaces = splitCSV(*namespaces)
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (cfg Config) Validate() error {
	if strings.TrimSpace(cfg.ListenAddress) == "" {
		return fmt.Errorf("listen address cannot be empty")
	}
	if strings.TrimSpace(cfg.KFPAPIURL) == "" {
		return fmt.Errorf("kfp api url cannot be empty")
	}
	parsed, err := url.ParseRequestURI(cfg.KFPAPIURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("kfp api url must be an absolute URL")
	}
	if cfg.PollInterval <= 0 {
		return fmt.Errorf("poll interval must be greater than zero")
	}
	if cfg.PageSize <= 0 {
		return fmt.Errorf("page size must be greater than zero")
	}
	if cfg.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be greater than zero")
	}
	if cfg.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be greater than zero")
	}
	return nil
}

func envInt(lookupEnv func(string) (string, bool), key string, fallback int) (int, error) {
	value, ok := lookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer: %w", key, err)
	}
	return parsed, nil
}

func (cfg Config) NamespaceMode() string {
	if len(cfg.Namespaces) == 0 {
		return "all-visible"
	}
	return "restricted"
}

func envString(lookupEnv func(string) (string, bool), key, fallback string) string {
	if value, ok := lookupEnv(key); ok {
		return value
	}
	return fallback
}

func envDuration(lookupEnv func(string) (string, bool), key string, fallback time.Duration) (time.Duration, error) {
	value, ok := lookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", key, err)
	}
	return duration, nil
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
