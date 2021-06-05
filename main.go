package main

import (
	// 公共引入
	_ "github.com/lecex/core/plugins"
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/util/log"

	// 执行数据迁移
	"github.com/lecex/pay/config"
	"github.com/lecex/pay/handler"
	_ "github.com/lecex/pay/providers/migrations"
)

func main() {
	var Conf = config.Conf
	service := micro.NewService(
		micro.Name(Conf.Name),
		micro.Version(Conf.Version),
	)
	service.Init()
	// 注册服务
	h := handler.Handler{
		Server: service.Server(),
	}
	h.Register()
	// Run the server
	log.Fatal("serviser run ... Version:" + Conf.Version)
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
