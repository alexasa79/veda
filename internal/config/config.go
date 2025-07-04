package config

type Config struct {
	ScrollOffset int
}

func New() *Config {
	return &Config{
		ScrollOffset: 10,
	}
}
