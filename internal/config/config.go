package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
)

// Config holds all centralized system parameters
type Config struct {
	ServerHost string
	ServerPort string
	WorldPath  string

	MaxConnections                int
	MaxConnectionsPerIP           int
	MaxConnectionAttemptsPerIP    int
	MaxUnauthenticatedConnections int
}

func (c *Config) ServerAddr() string {
	if c.ServerHost == "" {
		return ":" + c.ServerPort
	}
	return fmt.Sprintf("%s:%s", c.ServerHost, c.ServerPort)
}

// LoadConfig fetches settings from flags or environment variables
func LoadConfig() *Config {
	// 1. Set fallback default values
	defaultHost := ""
	defaultPort := "8090"
	defaultWorld := "data/world.json"
	defaultMaxConnections := envInt("TAP_MAX_CONNECTIONS", 100)
	defaultMaxConnectionsPerIP := envInt("TAP_MAX_CONNECTIONS_PER_IP", 5)
	defaultMaxConnectionAttemptsPerIP := envInt("TAP_MAX_CONNECTION_ATTEMPTS_PER_IP", 10)
	defaultMaxUnauthenticatedConnections := envInt("TAP_MAX_UNAUTHENTICATED_CONNECTIONS", 20)

	if envHost := os.Getenv("TAP_SERVER_HOST"); envHost != "" {
		defaultHost = envHost
	}
	// 2. Check for environment variables override (useful for Docker/CI)
	if envPort := os.Getenv("TAP_SERVER_PORT"); envPort != "" {
		defaultPort = envPort
	}

	// 3. Define CLI flags to allow execution runtime changes
	host := flag.String("host", defaultHost, "Server IP or hostname (used by clients)")
	port := flag.String("port", defaultPort, "TCP port for the game server to listen/connect")
	world := flag.String("world", defaultWorld, "Path to the world configuration json asset")
	maxConnections := flag.Int("max-connections", defaultMaxConnections, "Maximum simultaneous TCP connections")
	maxConnectionsPerIP := flag.Int("max-connections-per-ip", defaultMaxConnectionsPerIP, "Maximum simultaneous TCP connections from one IP")
	maxConnectionAttemptsPerIP := flag.Int("max-connection-attempts-per-ip", defaultMaxConnectionAttemptsPerIP, "Maximum TCP connection attempts per IP per minute")
	maxUnauthenticatedConnections := flag.Int("max-unauthenticated-connections", defaultMaxUnauthenticatedConnections, "Maximum simultaneous unauthenticated TCP connections")

	// flag.Parse() will be triggered at main invocation boundary
	flag.Parse()

	return &Config{
		ServerHost:                    *host,
		ServerPort:                    *port,
		WorldPath:                     *world,
		MaxConnections:                *maxConnections,
		MaxConnectionsPerIP:           *maxConnectionsPerIP,
		MaxConnectionAttemptsPerIP:    *maxConnectionAttemptsPerIP,
		MaxUnauthenticatedConnections: *maxUnauthenticatedConnections,
	}
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		log.Printf("Invalid %s value %q; using default %d", name, value, fallback)
		return fallback
	}
	return parsed
}
