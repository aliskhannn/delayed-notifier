package config

type Config struct {
	HTTPPort string   `yaml:"http_port"`
	Database Database `yaml:"database"`
}

type Database struct {
	Host    string
	Port    string
	User    string
	Pass    string
	Name    string
	SSLMode string `yaml:"ssl_mode"`
}
