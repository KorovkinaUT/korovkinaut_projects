package config

type RateLimitConfig struct {
	RPS   float64 `envconfig:"HTTP_RATE_LIMIT_RPS" default:"10"`
	Burst int     `envconfig:"HTTP_RATE_LIMIT_BURST" default:"20"`
}

func (c *RateLimitConfig) Validate() error {
	if c.RPS <= 0 {
		c.RPS = 10
	}

	if c.Burst <= 0 {
		c.Burst = 20
	}

	return nil
}
