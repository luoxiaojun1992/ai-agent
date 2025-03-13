package mcp

type IClient interface {
}

type Config struct {
	Host string
}

type Client struct {
	config *Config
}
