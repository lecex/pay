package handler

import (
	"context"
	"encoding/json"
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
	Config repository.Config
	Repo   repository.Order
	Alipay *service.Alipay
	Wechat *service.Wechat
	Order  *orderPB.Order
}

// UserConfig 用户配置
func (srv *Pay) UserConfig(order *pd.Order) (*configPB.Config, error) {
	config := &configPB.Config{}
	if order.StoreId != "" {
		config.Id = order.StoreId
	}
	if order.StoreName != "" {
		config.StoreName = order.StoreName
	}
	err := srv.Config.Get(config)
	return config, err
}

// HanderOrder 处理订单
func (srv *Pay) HanderOrder(order *pd.Order) (err error) {
	srv.Order = &orderPB.Order{
		StoreId:     order.StoreId,     // 商户门店编号 收款账号ID userID
		Method:      order.Method,      // 付款方式 [支付宝、微信、银联等]
		AuthCode:    order.AuthCode,    // 付款码
		Title:       order.Title,       // 订单标题
		TotalAmount: order.TotalAmount, // 订单总金额
		OrderNo:     order.OrderNo,     // 订单编号
		OperatorId:  order.OperatorId,  // 商户操作员编号
		TerminalId:  order.TerminalId,  // 商户机具终端编号
		Stauts:      0,                 // 订单状态 默认状态未付款
	}
	err = srv.Repo.StoreIdAndOrderNoGet(srv.Order)
	if srv.Order.StoreId != order.StoreId || srv.Order.OrderNo != order.OrderNo || srv.Order.Method != order.Method || srv.Order.AuthCode != order.AuthCode || srv.Order.TotalAmount != order.TotalAmount {
		return errors.New("上报订单已存在,但数据校验失败")
	}
	if err == gorm.ErrRecordNotFound {
		err = srv.Repo.Create(srv.Order)
	}
	return err
}

// AopF2F 商家扫用户付款码
func (srv *Pay) AopF2F(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	config, err := srv.UserConfig(req.Order)
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
	if srv.Order.Stauts == 1 {
		res.Valid = true
		return err // 支付成功返回
	}
	if srv.Order.Stauts == -1 {
		res.Valid = false
		return fmt.Errorf("订单已关闭")
	}
	switch req.Order.Method {
	case "alipay":
		srv.Alipay.NewClient(map[string]string{
			"AppId":                config.Alipay.AppId,
			"PrivateKey":           config.Alipay.PrivateKey,
			"AliPayPublicKey":      config.Alipay.AliPayPublicKey,
			"AppAuthToken":         config.Alipay.AppAuthToken,
			"SysServiceProviderId": config.Alipay.SysServiceProviderId,
			"SignType":             config.Alipay.SignType,
		}, config.Alipay.Sandbox)
		res.Valid, err = srv.Alipay.AopF2F(req.Order)
		if err != nil {
			srv.alipayError(err)
			res.Valid = false
			return err
		}
		if res.Valid {
			srv.Order.Stauts = 1
		}
		err = srv.Repo.Update(srv.Order)
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
			srv.wechatError(err)
			res.Valid = false
			return err
		}
		if res.Valid {
			srv.Order.Stauts = 1
		}
		err = srv.Repo.Update(srv.Order)
		if err != nil {
			res.Valid = false
			return fmt.Errorf("订单状态更新失败:%s", err)
		}
		return err
	}
	return err
}

// alipayError 支付宝错误
func (srv *Pay) alipayError(err error) (e error) {
	s := map[string]string{}
	e = json.Unmarshal([]byte(err.Error()), &s)
	if e != nil {
		return e
	}
	if s["sub_code"] == "ACQ.TRADE_HAS_CLOSE" {
		srv.Order.Stauts = -1
		srv.Repo.Update(srv.Order)
	}
	return e
}

// wechatError 微信错误
func (srv *Pay) wechatError(err error) (e error) {
	s := map[string]string{}
	e = json.Unmarshal([]byte(err.Error()), &s)
	if e != nil {
		return e
	}
	if s["err_code"] == "ORDERCLOSED" {
		srv.Order.Stauts = -1
		srv.Repo.Update(srv.Order)
	}
	return e
}
