package config_test

import (
	"flag"
	"os"
	"testing"

	"the-answer-protocol/internal/config"
)

func TestServerAddrUsesHostWhenConfigured(t *testing.T) {
	cfg := &config.Config{ServerHost: "127.0.0.1", ServerPort: "9000"}
	if got, want := cfg.ServerAddr(), "127.0.0.1:9000"; got != want {
		t.Fatalf("ServerAddr() = %q, want %q", got, want)
	}
}

func TestServerAddrUsesAnyInterfaceWithoutHost(t *testing.T) {
	cfg := &config.Config{ServerPort: "8090"}
	if got, want := cfg.ServerAddr(), ":8090"; got != want {
		t.Fatalf("ServerAddr() = %q, want %q", got, want)
	}
}

func TestLoadConfigReadsEnvironmentAndFlags(t *testing.T) {
	previousFlags := flag.CommandLine
	previousArgs := os.Args
	t.Cleanup(func() {
		flag.CommandLine = previousFlags
		os.Args = previousArgs
	})

	t.Setenv("TAP_SERVER_HOST", "env.example")
	t.Setenv("TAP_SERVER_PORT", "8100")
	t.Setenv("TAP_MAX_CONNECTIONS", "75")
	flag.CommandLine = flag.NewFlagSet("config-test", flag.ContinueOnError)
	os.Args = []string{
		"config-test",
		"-host=flag.example",
		"-port=8200",
		"-world=/tmp/test-world.json",
		"-max-connections=80",
		"-max-connections-per-ip=4",
	}

	cfg := config.LoadConfig()
	if cfg.ServerHost != "flag.example" || cfg.ServerPort != "8200" ||
		cfg.WorldPath != "/tmp/test-world.json" || cfg.MaxConnections != 80 ||
		cfg.MaxConnectionsPerIP != 4 || cfg.MaxConnectionAttemptsPerIP != 10 ||
		cfg.MaxUnauthenticatedConnections != 20 {
		t.Fatalf("LoadConfig() = %+v, want flag overrides", cfg)
	}
}
