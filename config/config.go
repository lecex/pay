package config

// Config 配置
type Config struct {
	Service string
	Version string
}

// Conf 配置
var Conf Config = Config{
	Service: "pay",
	Version: "latest",
}
