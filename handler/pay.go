package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
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
func (srv *Pay) UserConfig(userID string) (*configPB.Config, error) {
	config := &configPB.Config{
		Id: userID,
	}
	err := srv.Config.Get(config)
	return config, err
}

// HanderOrder 处理订单
func (srv *Pay) HanderOrder(order *pd.Order) (stauts bool, err error) {
	resOrder := &orderPB.Order{
		Id:          order.Id,          // 订单编号 UUID 前端生产全局唯一
		StoreId:     order.StoreId,     // 商户门店编号 收款账号ID userID
		Method:      order.Method,      // 付款方式 [支付宝、微信、银联等]
		AuthCode:    order.AuthCode,    // 付款码
		Title:       order.Title,       // 订单标题
		TotalAmount: order.TotalAmount, // 订单总金额
		OperatorId:  order.OperatorId,  // 商户操作员编号
		TerminalId:  order.TerminalId,  // 商户机具终端编号
		Stauts:      false,             // 订单状态 默认状态未付款
	}
	err = srv.Order.Get(resOrder)
	if resOrder.StoreId != order.StoreId || resOrder.Method != order.Method || resOrder.AuthCode != order.AuthCode || resOrder.TotalAmount != order.TotalAmount {
		return false, errors.New("上报订单已存在,但数据校验失败")
	}
	if err == gorm.ErrRecordNotFound {
		err = srv.Order.Create(resOrder)
	}
	stauts = resOrder.Stauts
	return stauts, err
}

// AopF2F 商家扫用户付款码
func (srv *Pay) AopF2F(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	fmt.Println("AopF2F:%s", time.Now())
	config, err := srv.UserConfig(req.Order.StoreId)
	fmt.Println("UserConfig:%s", time.Now())
	if err != nil {
		res.Valid = false
		return fmt.Errorf("查询配置信息失败:%s", err)
	}
	res.Valid, err = srv.HanderOrder(req.Order) //创建订单返回订单ID
	fmt.Println("HanderOrder:%s", time.Now())
	if err != nil {
		res.Valid = false
		return fmt.Errorf("创建订单失败:%s", err)
	}
	if res.Valid {
		return err // 支付成功返回
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
		fmt.Println("Alipay:%s", time.Now())
		fmt.Println("Alipay:%s 1 %s", res.Valid, err)
		if err != nil {
			res.Valid = false
			return fmt.Errorf("支付失败:%s", err)
		}
		err = srv.Order.Update(&orderPB.Order{
			Id:     req.Order.Id,
			Stauts: true, // 订单状态 默认状态未付款
		})
		fmt.Println("Update:%s", time.Now())
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
		err = srv.Order.Update(&orderPB.Order{
			Id:     req.Order.Id,
			Stauts: true, // 订单状态 默认状态未付款
		})
		if err != nil {
			res.Valid = false
			return fmt.Errorf("订单状态更新失败:%s", err)
		}
		return err
	}
	return err
}
