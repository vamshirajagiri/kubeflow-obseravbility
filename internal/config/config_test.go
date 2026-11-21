package config

import (
	"reflect"
	"testing"
	"time"
)

func TestFromArgsUsesDefaults(t *testing.T) {
	cfg, err := FromArgs(nil, mapEnv(map[string]string{envKFPAPIURL: "http://kfp.example"}))
	if err != nil {
		t.Fatalf("FromArgs returned error: %v", err)
	}

	if cfg.ListenAddress != ":8080" {
		t.Fatalf("ListenAddress = %q, want :8080", cfg.ListenAddress)
	}
	if cfg.NamespaceMode() != "all-visible" {
		t.Fatalf("NamespaceMode = %q, want all-visible", cfg.NamespaceMode())
	}
	if cfg.PollInterval != time.Minute {
		t.Fatalf("PollInterval = %s, want 1m", cfg.PollInterval)
	}
}

func TestFromArgsParsesEnvironment(t *testing.T) {
	env := map[string]string{
		envListenAddress: ":9090",
		envKFPAPIURL:     "http://ml-pipeline.kubeflow.svc:8888",
		envNamespaces:    "team-a, team-b",
		envPollInterval:  "30s",
		envPageSize:      "250",
	}

	cfg, err := FromArgs(nil, mapEnv(env))
	if err != nil {
		t.Fatalf("FromArgs returned error: %v", err)
	}

	if cfg.ListenAddress != ":9090" {
		t.Fatalf("ListenAddress = %q, want :9090", cfg.ListenAddress)
	}
	if cfg.KFPAPIURL != "http://ml-pipeline.kubeflow.svc:8888" {
		t.Fatalf("KFPAPIURL = %q", cfg.KFPAPIURL)
	}
	if !reflect.DeepEqual(cfg.Namespaces, []string{"team-a", "team-b"}) {
		t.Fatalf("Namespaces = %#v", cfg.Namespaces)
	}
	if cfg.NamespaceMode() != "restricted" {
		t.Fatalf("NamespaceMode = %q, want restricted", cfg.NamespaceMode())
	}
	if cfg.PageSize != 250 {
		t.Fatalf("PageSize = %d, want 250", cfg.PageSize)
	}
}

func TestFromArgsFlagsOverrideEnvironment(t *testing.T) {
	env := map[string]string{
		envListenAddress: ":9090",
		envNamespaces:    "team-a",
		envPollInterval:  "30s",
	}

	cfg, err := FromArgs([]string{
		"-listen-address=:7070",
		"-kfp-api-url=http://override.example",
		"-namespaces=team-c,team-d",
		"-poll-interval=45s",
	}, mapEnv(env))
	if err != nil {
		t.Fatalf("FromArgs returned error: %v", err)
	}

	if cfg.ListenAddress != ":7070" {
		t.Fatalf("ListenAddress = %q, want :7070", cfg.ListenAddress)
	}
	if !reflect.DeepEqual(cfg.Namespaces, []string{"team-c", "team-d"}) {
		t.Fatalf("Namespaces = %#v", cfg.Namespaces)
	}
	if cfg.PollInterval != 45*time.Second {
		t.Fatalf("PollInterval = %s, want 45s", cfg.PollInterval)
	}
}

func TestFromArgsRejectsInvalidPollInterval(t *testing.T) {
	_, err := FromArgs([]string{"-kfp-api-url=http://kfp.example", "-poll-interval=0s"}, emptyEnv)
	if err == nil {
		t.Fatal("FromArgs returned nil error")
	}
}

func TestFromArgsRejectsInvalidEnvironmentPollInterval(t *testing.T) {
	env := map[string]string{
		envPollInterval: "60seconds",
	}

	_, err := FromArgs(nil, mapEnv(env))
	if err == nil {
		t.Fatal("FromArgs returned nil error")
	}
}

func TestFromArgsRequiresKFPAPIURL(t *testing.T) {
	_, err := FromArgs(nil, emptyEnv)
	if err == nil {
		t.Fatal("FromArgs returned nil error")
	}
}

func emptyEnv(string) (string, bool) {
	return "", false
}

func mapEnv(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
