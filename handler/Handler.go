package handler

import (
	server "github.com/micro/go-micro/v2/server"

	db "github.com/lecex/pay/providers/database"

	configPB "github.com/lecex/pay/proto/config"
	orderPB "github.com/lecex/pay/proto/order"
	tradePB "github.com/lecex/pay/proto/trade"

	"github.com/lecex/pay/service/repository"
	"github.com/lecex/pay/service/trade"
)

// Handler 注册方法
type Handler struct {
	Server server.Server
}

// Register 注册
func (srv *Handler) Register() {
	configPB.RegisterConfigsHandler(srv.Server, srv.Config()) // 配置服务实现
	orderPB.RegisterOrdersHandler(srv.Server, srv.Order())    // 订单服务实现
	tradePB.RegisterTradesHandler(srv.Server, srv.Trade())    // 支付服务实现
}

// Config 订单管理服务实现
func (srv *Handler) Config() *Config {
	return &Config{&repository.ConfigRepository{db.DB}}
}

// Order 订单管理服务实现
func (srv *Handler) Order() *Order {
	return &Order{&repository.OrderRepository{db.DB}}
}

// Trade 订单管理服务实现
func (srv *Handler) Trade() *Trade {
	return &Trade{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		Alipay: &trade.Alipay{},
		Wechat: &trade.Wechat{},
		Icbc:   &trade.Icbc{},
	}
}
