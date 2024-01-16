package suave

type Config struct {
	Enabled bool
}

var DefaultConfig = Config{
	Enabled: false,
}
