package config

import (
	"flag"
	"fmt"
	"os"
)

// Config holds all centralized system parameters
type Config struct {
	ServerHost string
	ServerPort string
	WorldPath  string
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

	// flag.Parse() will be triggered at main invocation boundary
	flag.Parse()

	return &Config{
		ServerHost: *host,
		ServerPort: *port,
		WorldPath:  *world,
	}
}
