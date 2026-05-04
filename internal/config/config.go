package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Schema      string            `yaml:"schema"`
	Output      string            `yaml:"output"`
	Endpoint    string            `yaml:"endpoint"`
	Credentials map[string]string `yaml:"credentials"`
	Headers     map[string]string `yaml:"headers"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	cfg.interpolateHeaders()

	return &cfg, nil
}

// interpolateHeaders
func (c *Config) interpolateHeaders() {
	if c.Credentials == nil || c.Headers == nil {
		return
	}

	for headerName, headerValue := range c.Headers {
		interpolatedValue := headerValue

		// Go through each credential and replace placeholders in the header value
		for credKey, credVal := range c.Credentials {
			placeholder := fmt.Sprintf("{{credentials.%s}}", credKey)
			// Replace all occurrences of the placeholder with the actual credential value
			interpolatedValue = strings.ReplaceAll(interpolatedValue, placeholder, credVal)
		}
		c.Headers[headerName] = interpolatedValue
	}
}
