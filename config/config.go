package config

import "github.com/lecex/core/env"

// Config 配置
type Config struct {
	Service string
	Version string
}

// Conf 配置
var Conf Config = Config{
	Service: env.Getenv("MICRO_API_NAMESPACE", "go.micro.srv") + "pay",
	Version: "latest",
}
