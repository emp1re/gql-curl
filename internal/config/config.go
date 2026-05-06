package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// MaxDepth defines how deep the generator will expand nested types when building the selection set.
var (
	MaxDepth               int = 3
	DocumentExtensions         = []string{".graphql", ".graphqls"}
	EnableLogging          bool
	LogFileTimestampFormat string = "2006-01-02"
)

type Config struct {
	Schema          string            `yaml:"schema"`
	DocumentExc     []string          `yaml:"document_extensions"`
	Output          string            `yaml:"output"`
	TimestampFormat string            `yaml:"log_file_timestamp_format"`
	Endpoint        string            `yaml:"endpoint"`
	Environment     map[string]string `yaml:"environment"`
	Headers         map[string]string `yaml:"headers"`
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
	// Set default document extensions if not provided
	DocumentExtensions = cfg.DocumentExc

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

			if envKey == "MAX_DEPTH" {
				MaxDepth, _ = strconv.Atoi(envVal)
			}
			if envKey == "ENABLE_LOGGING" {
				EnableLogging, _ = strconv.ParseBool(envVal)
			}
			if envKey == "LOG_FILE_TIMESTAMP_FORMAT" {
				LogFileTimestampFormat = envVal
			}

			placeholder := fmt.Sprintf("{{environment.%s}}", envKey)
			interpolatedValue = strings.ReplaceAll(interpolatedValue, placeholder, envVal)
		}

		c.Headers[headerName] = interpolatedValue
	}
}
