package config

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// MaxDepth defines how deep the generator will expand nested types when building the selection set.
var (
	DefaultMaxDepth           = 3
	MaxDepth                  = DefaultMaxDepth
	DefaultDocumentExtensions = []string{".graphql", ".graphqls"}
	DocumentExtensions        = append([]string(nil), DefaultDocumentExtensions...)
)

type StringList []string

func (s *StringList) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		if value.Tag == "!!null" {
			*s = nil
			return nil
		}
		*s = []string{value.Value}
	case yaml.SequenceNode:
		var items []string
		if err := value.Decode(&items); err != nil {
			return err
		}
		*s = items
	default:
		return fmt.Errorf("expected string or list of strings")
	}

	return nil
}

type StringMap map[string]string

func (m *StringMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Tag == "!!null" {
		*m = nil
		return nil
	}
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("expected map")
	}

	result := make(map[string]string, len(value.Content)/2)
	for i := 0; i < len(value.Content); i += 2 {
		key := value.Content[i].Value
		val := value.Content[i+1]
		if val.Kind != yaml.ScalarNode {
			return fmt.Errorf("environment.%s must be a scalar value", key)
		}

		result[key] = val.Value
	}

	*m = result
	return nil
}

type SchemaConfig struct {
	Path      StringList        `yaml:"path"`
	Endpoint  string            `yaml:"endpoint"`
	AuthToken string            `yaml:"auth_token"`
	Headers   map[string]string `yaml:"headers"`
}

type NamedSchema struct {
	Name   string
	Config SchemaConfig
}

type Config struct {
	Schemas     map[string]SchemaConfig `yaml:"schemas"`
	DocumentExc []string                `yaml:"document_extensions"`
	Environment StringMap               `yaml:"environment"`
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

	// Set default document extensions if not provided.
	if len(cfg.DocumentExc) > 0 {
		DocumentExtensions = append([]string(nil), cfg.DocumentExc...)
	} else {
		DocumentExtensions = append([]string(nil), DefaultDocumentExtensions...)
		cfg.DocumentExc = append([]string(nil), DocumentExtensions...)
	}

	if err := cfg.applyEnvironment(); err != nil {
		return nil, err
	}
	if err := cfg.interpolateSchemas(); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) SchemaNames() []string {
	names := make([]string, 0, len(c.Schemas))
	for name := range c.Schemas {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

func (c *Config) SelectedSchemas(schemaName string) ([]NamedSchema, error) {
	if schemaName != "" {
		schema, ok := c.Schemas[schemaName]
		if !ok {
			return nil, fmt.Errorf("schema %q is not configured", schemaName)
		}

		return []NamedSchema{{Name: schemaName, Config: schema}}, nil
	}

	names := c.SchemaNames()
	schemas := make([]NamedSchema, 0, len(names))
	for _, name := range names {
		schemas = append(schemas, NamedSchema{Name: name, Config: c.Schemas[name]})
	}

	return schemas, nil
}

func (c *Config) applyEnvironment() error {
	MaxDepth = DefaultMaxDepth
	if c.Environment == nil {
		return nil
	}

	if rawMaxDepth, ok := c.Environment["MAX_DEPTH"]; ok {
		maxDepth, err := strconv.Atoi(rawMaxDepth)
		if err != nil {
			return fmt.Errorf("environment.MAX_DEPTH must be an integer: %w", err)
		}
		MaxDepth = maxDepth
	}

	return nil
}

func (c *Config) interpolateSchemas() error {
	for schemaName, schema := range c.Schemas {
		if schema.Headers == nil {
			continue
		}

		for headerName, headerValue := range schema.Headers {
			interpolatedValue := strings.ReplaceAll(headerValue, "{{auth_token}}", schema.AuthToken)
			for envKey, envVal := range c.Environment {
				placeholder := fmt.Sprintf("{{environment.%s}}", envKey)
				interpolatedValue = strings.ReplaceAll(interpolatedValue, placeholder, envVal)
			}

			schema.Headers[headerName] = interpolatedValue
		}

		c.Schemas[schemaName] = schema
	}

	return nil
}

func (c *Config) validate() error {
	if len(c.Schemas) == 0 {
		return fmt.Errorf("schemas is required")
	}

	for schemaName, schema := range c.Schemas {
		if len(schema.Path) == 0 {
			return fmt.Errorf("schemas.%s.path is required", schemaName)
		}
		if strings.TrimSpace(schema.Endpoint) == "" {
			return fmt.Errorf("schemas.%s.endpoint is required", schemaName)
		}
	}

	return nil
}
