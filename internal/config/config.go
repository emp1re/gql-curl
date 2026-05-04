package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Schema      string            `yaml:"schema"`
	Output      string            `yaml:"output"`
	Endpoint    string            `yaml:"endpoint"`
	Environment map[string]string `yaml:"environment"`
	Headers     map[string]string `yaml:"headers"`
}

func LoadConfig(path string) (*Config, error) {

	_ = godotenv.Load()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file failed: %w", err)
	}

	expandedData := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config failed: %w", err)
	}

	// {{environment.KEY}}
	cfg.interpolateHeaders()

	return &cfg, nil
}

// interpolateHeaders
func (c *Config) interpolateHeaders() {
	if c.Environment == nil || c.Headers == nil {
		return
	}

	for headerName, headerValue := range c.Headers {
		interpolatedValue := headerValue

		for envKey, envVal := range c.Environment {
			placeholder := fmt.Sprintf("{{environment.%s}}", envKey)
			interpolatedValue = strings.ReplaceAll(interpolatedValue, placeholder, envVal)
		}

		c.Headers[headerName] = interpolatedValue
	}
}
