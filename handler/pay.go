package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/jinzhu/gorm"
	configPB "github.com/lecex/pay/proto/config"
	orderPB "github.com/lecex/pay/proto/order"
	pd "github.com/lecex/pay/proto/pay"
	"github.com/lecex/pay/service"
	"github.com/lecex/pay/service/repository"
)

// Pay 支付结构
type Pay struct {
	Config  repository.Config
	Order   repository.Order
	Alipay  *service.Alipay
	Wechat  *service.Wechat
	OrderDB *orderPB.Order
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
func (srv *Pay) HanderOrder(order *pd.Order) (err error) {
	srv.OrderDB = &orderPB.Order{
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
	err = srv.Order.Get(srv.OrderDB)
	if srv.OrderDB.StoreId != order.StoreId || srv.OrderDB.Method != order.Method || srv.OrderDB.AuthCode != order.AuthCode || srv.OrderDB.TotalAmount != order.TotalAmount {
		return errors.New("上报订单已存在,但数据校验失败")
	}
	if err == gorm.ErrRecordNotFound {
		err = srv.Order.Create(srv.OrderDB)
	}
	return err
}

// AopF2F 商家扫用户付款码
func (srv *Pay) AopF2F(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	config, err := srv.UserConfig(req.Order.StoreId)
	if err != nil {
		res.Valid = false
		return fmt.Errorf("查询配置信息失败:%s", err)
	}
	if !config.Stauts {
		return fmt.Errorf("支付功能被禁用！请联系管理员。")
	}
	err = srv.HanderOrder(req.Order) //创建订单返回订单ID
	if err != nil {
		res.Valid = false
		return fmt.Errorf("创建订单失败:%s", err)
	}
	if srv.OrderDB.Stauts {
		return err // 支付成功返回
	}
	switch req.Order.Method {
	case "alipay":
		srv.Alipay.NewClient(map[string]string{
			"AppId":           config.Alipay.AppId,
			"PrivateKey":      config.Alipay.PrivateKey,
			"AliPayPublicKey": config.Alipay.AliPayPublicKey,
			"SignType":        config.Alipay.SignType,
		}, config.Alipay.Sandbox)
		res.Valid, err = srv.Alipay.AopF2F(req.Order)
		if err != nil {
			res.Valid = false
			return err
		}
		srv.OrderDB.Stauts = res.Valid
		err = srv.Order.Update(srv.OrderDB)
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
		}, config.Wechat.Sandbox)
		res.Valid, err = srv.Wechat.AopF2F(req.Order)
		if err != nil {
			res.Valid = false
			return err
		}
		srv.OrderDB.Stauts = res.Valid
		err = srv.Order.Update(srv.OrderDB)
		if err != nil {
			res.Valid = false
			return fmt.Errorf("订单状态更新失败:%s", err)
		}
		return err
	}
	return err
}
