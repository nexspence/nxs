package config

type Config struct{}

func Load(ctx string) (*Config, error) { return &Config{}, nil }
func (c *Config) CurrentURL() string   { return "" }
func (c *Config) CurrentToken() string { return "" }
