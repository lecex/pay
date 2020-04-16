package handler

import (
	server "github.com/micro/go-micro/v2/server"

	db "github.com/lecex/pay/providers/database"

	configPB "github.com/lecex/pay/proto/config"
	notifyPB "github.com/lecex/pay/proto/notify"
	orderPB "github.com/lecex/pay/proto/order"
	payPB "github.com/lecex/pay/proto/pay"

	"github.com/lecex/pay/service"
	"github.com/lecex/pay/service/repository"
)

// Handler 注册方法
type Handler struct {
	Server server.Server
}

// Register 注册
func (srv *Handler) Register() {
	configPB.RegisterConfigsHandler(srv.Server, srv.Config()) // 配置服务实现
	orderPB.RegisterOrdersHandler(srv.Server, srv.Order())    // 订单服务实现
	payPB.RegisterPaysHandler(srv.Server, srv.Pay())          // 支付服务实现
	notifyPB.RegisterNotifyHandler(srv.Server, srv.Notify())  // 支付异步通知服务实现
}

// Config 订单管理服务实现
func (srv *Handler) Config() *Config {
	return &Config{&repository.ConfigRepository{db.DB}}
}

// Order 订单管理服务实现
func (srv *Handler) Order() *Order {
	return &Order{&repository.OrderRepository{db.DB}}
}

// Pay 订单管理服务实现
func (srv *Handler) Pay() *Pay {
	return &Pay{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		Alipay: &service.Alipay{},
		Wechat: &service.Wechat{},
	}
}

// Notify 异步通知服务实现
func (srv *Handler) Notify() *Notify {
	return &Notify{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		alipay: &service.Alipay{},
		wechat: &service.Wechat{},
	}
}
