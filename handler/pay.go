package handler

import (
	"context"
	"fmt"

	configPB "github.com/lecex/pay/proto/config"
	orderPB "github.com/lecex/pay/proto/order"
	pd "github.com/lecex/pay/proto/pay"
	"github.com/lecex/pay/service"
	"github.com/lecex/pay/service/repository"
)

// Pay 支付结构
type Pay struct {
	Config repository.Config
	Order  repository.Order
	Alipay *service.Alipay
	Wechat *service.Wechat
}

// UserConfig 用户配置
func (srv *Pay) UserConfig(userId string) (config *configPB.Config, err error) {
	return srv.Config.Get(&configPB.Config{
		Id: userId,
	})
}

// CreateOrder 创建订单
func (srv *Pay) CreateOrder(order *pd.Order) (orderId string, err error) {
	res, err := srv.Order.Create(&orderPB.Order{
		StoreId:     order.StoreId,     // 商户门店编号 收款账号ID userID
		Method:      order.Method,      // 付款方式 [支付宝、微信、银联等]
		AuthCode:    order.AuthCode,    // 付款码
		Title:       order.Title,       // 订单标题
		OrderSn:     order.OrderSn,     // 订单编号
		TotalAmount: order.TotalAmount, // 订单总金额
		OperatorId:  order.OperatorId,  // 商户操作员编号
		TerminalId:  order.TerminalId,  // 商户机具终端编号
		Stauts:      false,             // 订单状态 默认状态未付款
	})
	orderId = res.Id
	return
}

// UpdataOrder 更新订单状态
func (srv *Pay) UpdataOrder(orderId string, stauts bool) (err error) {
	_, err = srv.Order.Update(&orderPB.Order{
		Id:     orderId,
		Stauts: stauts, // 订单状态 默认状态未付款
	})
	return
}

// AopF2F 商家扫用户付款码
func (srv *Pay) AopF2F(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	config, err := srv.UserConfig(req.Order.StoreId)
	if err != nil {
		res.Valid = false
		return fmt.Errorf("查询配置信息失败:%s", err)
	}
	orderId, err := srv.CreateOrder(req.Order) //创建订单返回订单ID
	if err != nil {
		res.Valid = false
		return fmt.Errorf("创建订单失败:%s", err)
	}
	switch req.Order.Method {
	case "alipay":
		srv.Alipay.NewClient(map[string]string{
			"AppId":           config.Alipay.AppId,
			"PrivateKey":      config.Alipay.PrivateKey,
			"AliPayPublicKey": config.Alipay.AliPayPublicKey,
			"SignType":        config.Alipay.SignType,
		}, true)
		res.Valid, err = srv.Alipay.AopF2F(req.Order)
		if err != nil {
			res.Valid = false
			return fmt.Errorf("支付失败:%s", err)
		}
		err = srv.UpdataOrder(orderId, res.Valid)
		if err != nil {
			res.Valid = false
			return fmt.Errorf("订单状态更新失败:%s", err)
		}
		return err
	case "wechat":
		srv.Wechat.NewClient(map[string]string{
			"AppId":    config.Wechat.AppId,
			"MchId":    config.Wechat.MchId,
			"ApiKey":   config.Wechat.ApiKey,
			"SubAppId": config.Wechat.SubAppId,
			"SubMchId": config.Wechat.SubMchId,
		}, true)
		res.Valid, err = srv.Wechat.AopF2F(req.Order)
		if err != nil {
			res.Valid = false
			return fmt.Errorf("支付失败:%s", err)
		}
		err = srv.UpdataOrder(orderId, res.Valid)
		if err != nil {
			res.Valid = false
			return fmt.Errorf("订单状态更新失败:%s", err)
		}
		return err
	}
	return err
}
