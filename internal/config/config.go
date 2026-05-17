package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Context struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

type Config struct {
	CurrentContext string             `yaml:"current_context"`
	Contexts       map[string]Context `yaml:"contexts"`

	activeContext string
	envURL        string
	envToken      string
}

func Load(contextOverride string) (*Config, error) {
	cfgPath := os.Getenv("NXS_CONFIG")
	if cfgPath == "" {
		home, _ := os.UserHomeDir()
		cfgPath = filepath.Join(home, ".config", "nxs", "config.yaml")
	}

	cfg := &Config{Contexts: map[string]Context{}}
	data, err := os.ReadFile(cfgPath)
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	switch {
	case contextOverride != "":
		cfg.activeContext = contextOverride
	case os.Getenv("NXS_CONTEXT") != "":
		cfg.activeContext = os.Getenv("NXS_CONTEXT")
	default:
		cfg.activeContext = cfg.CurrentContext
	}

	cfg.envURL = os.Getenv("NXS_URL")
	cfg.envToken = os.Getenv("NXS_TOKEN")
	return cfg, nil
}

func (c *Config) CurrentURL() string {
	if c.envURL != "" {
		return c.envURL
	}
	return c.Contexts[c.activeContext].URL
}

func (c *Config) CurrentToken() string {
	if c.envToken != "" {
		return c.envToken
	}
	return c.Contexts[c.activeContext].Token
}

func (c *Config) ActiveContext() string {
	return c.activeContext
}

func (c *Config) ListContexts() []string {
	names := make([]string, 0, len(c.Contexts))
	for name := range c.Contexts {
		names = append(names, name)
	}
	return names
}

func (c *Config) Save(cfgPath, contextName, url, token string) error {
	if cfgPath == "" {
		home, _ := os.UserHomeDir()
		cfgPath = filepath.Join(home, ".config", "nxs", "config.yaml")
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0700); err != nil {
		return err
	}
	if c.Contexts == nil {
		c.Contexts = map[string]Context{}
	}
	c.Contexts[contextName] = Context{URL: url, Token: token}
	c.CurrentContext = contextName
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(cfgPath, data, 0600)
}
