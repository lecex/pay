package database

import (
	"github.com/micro/go-micro/v2/util/log"

	"github.com/jinzhu/gorm"
	"github.com/lecex/core/env"
	tgorm "github.com/lecex/core/gorm"
)

// DB 管理包
var (
	// DB 连接
	DB *gorm.DB
)

func init() {
	var err error
	conf := &tgorm.Config{
		Driver: env.Getenv("DB_DRIVER", "mysql"),
		// Host 主机地址
		Host: env.Getenv("DB_HOST", "127.0.0.1"),
		// Port 主机端口
		Port: env.Getenv("DB_PORT", "3306"),
		// User 用户名
		User: env.Getenv("DB_USER", "root"),
		// Password 密码
		Password: env.Getenv("DB_PASSWORD", "root"),
		// DbName 数据库名称
		DbName: env.Getenv("DB_NAME", "pay"),
		// Charset 数据库编码
		Charset: env.Getenv("DB_CHARSET", "utf8"),
	}
	DB, err = tgorm.Connection(conf)
	if err != nil {
		log.Fatalf("connect error: %v\n", err)
	}
}
