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
	"github.com/micro/go-micro/v2/util/log"
)

// Pay 支付结构
type Pay struct {
	Config repository.Config
	Repo   repository.Order
	Alipay *service.Alipay
	Wechat *service.Wechat
	Order  *orderPB.Order
}

// Query 支付查询
func (srv *Pay) Query(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	config, err := srv.UserConfig(req.Order)
	if err != nil {
		log.Fatal("UserConfig.Query")
		log.Fatal(err, req)
		return fmt.Errorf("查询配置信息失败:%s", err)
	}
	err = srv.GetOrder(req.Order) //创建订单返回订单ID
	if err != nil {
		log.Fatal("GetOrder.Query")
		log.Fatal(err, req)
		return fmt.Errorf("查询订单失败:%s", err)
	}
	// if srv.Order.Stauts == 1 {
	// 	res.Valid = true
	// 	return err // 支付成功返回
	// }
	// if srv.Order.Stauts == -1 {
	// 	return fmt.Errorf("订单已关闭")
	// }
	switch srv.Order.Method {
	case "alipay":
		srv.newAlipayClient(config) //实例化支付宝连接
		data, err := srv.Alipay.Query(req.Order)
		if err != nil {
			log.Fatal("Alipay.Query")
			log.Fatal(err, req)
			return err
		}
		if data["code"].(string) == "10000" && data["msg"].(string) == "Success" && data["trade_status"] == "TRADE_SUCCESS" {
			srv.Order.Stauts = 1
			err = srv.Repo.Update(srv.Order)
			if err != nil {
				log.Fatal("Alipay.Query.Update.1")
				log.Fatal(err, req)
				return fmt.Errorf("订单状态更新失败:%s", err)
			}
			res.Valid = true
			return err
		}
		if data["trade_status"] == "TRADE_CLOSED" || data["trade_status"] == "TRADE_FINISHED" || data["sub_code"] == "ACQ.SYSTEM_ERROR" || data["sub_code"] == "ACQ.INVALID_PARAMETER" || data["sub_code"] == "ACQ.TRADE_NOT_EXIST" {
			srv.Order.Stauts = -1
			err = srv.Repo.Update(srv.Order)
			if err != nil {
				log.Fatal("Alipay.Query.Update.-1")
				log.Fatal(err, req)
			}
		}
		e, _ := data.Json() //无法正常返回时
		log.Fatal("Alipay.Query.data")
		log.Fatal(e, req)
		return fmt.Errorf(string(e))
	case "wechat":
		srv.newWechatClient(config) //实例化连微信接
		data, err := srv.Wechat.Query(req.Order)
		if err != nil {
			log.Fatal("Wechat.Query")
			log.Fatal(err, req)
			return err
		}
		if data["trade_state"] == "SUCCESS" {
			srv.Order.Stauts = 1
			err = srv.Repo.Update(srv.Order)
			if err != nil {
				log.Fatal("Wechat.Query.Update.1")
				log.Fatal(err, req)
				return fmt.Errorf("订单状态更新失败:%s", err)
			}
			res.Valid = true
			return err
		}
		if data["trade_state"] == "REFUND" || data["trade_state"] == "CLOSED" || data["trade_state"] == "REVOKED" || data["trade_state"] == "PAYERROR" {
			srv.Order.Stauts = -1
			err = srv.Repo.Update(srv.Order)
			if err != nil {
				log.Fatal("Wechat.Query.Update.-1")
				log.Fatal(err, req)
			}
		}
		e, _ := data.Json() //无法正常返回时
		log.Fatal("Wechat.Query.data")
		log.Fatal(e, req)
		return fmt.Errorf(string(e))
	}
	return err
}

// AopF2F 商家扫用户付款码
func (srv *Pay) AopF2F(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	config, err := srv.UserConfig(req.Order)
	if err != nil {
		log.Fatal("UserConfig.AopF2F")
		log.Fatal(err, req)
		return fmt.Errorf("查询配置信息失败:%s", err)
	}
	if !config.Stauts {
		return fmt.Errorf("支付功能被禁用！请联系管理员。")
	}
	err = srv.HanderOrder(req.Order) //创建订单返回订单ID
	if err != nil {
		log.Fatal("UserConfig.AopF2F")
		log.Fatal(err, req)
		return fmt.Errorf("创建订单失败:%s", err)
	}
	// if srv.Order.Stauts == 1 {
	// 	res.Valid = true
	// 	return err // 支付成功返回
	// }
	// if srv.Order.Stauts == -1 {
	// 	return fmt.Errorf("订单已关闭")
	// }
	switch req.Order.Method {
	case "alipay":
		srv.newAlipayClient(config) //实例化连支付宝接
		data, err := srv.Alipay.AopF2F(req.Order)
		if err != nil {
			log.Fatal("Alipay.AopF2F")
			log.Fatal(err, req)
			return err
		}
		if data["code"].(string) == "10000" && data["msg"].(string) == "Success" {
			srv.Order.Stauts = 1
			err = srv.Repo.Update(srv.Order)
			if err != nil {
				log.Fatal("Alipay.AopF2F.Update.1")
				log.Fatal(err, req)
				return fmt.Errorf("订单状态更新失败:%s", err)
			}
			res.Valid = false
			return err
		}
		e, _ := data.Json() //无法正常返回时
		log.Fatal("Alipay.AopF2F.data")
		log.Fatal(e, req)
		return fmt.Errorf(string(e))
	case "wechat":
		srv.newWechatClient(config) //实例化微信连接
		data, err := srv.Wechat.AopF2F(req.Order)
		if err != nil {
			log.Fatal("Wechat.AopF2F")
			log.Fatal(err, req)
			return err
		}
		if data["result_code"] == "SUCCESS" {
			srv.Order.Stauts = 1
			err = srv.Repo.Update(srv.Order)
			if err != nil {
				log.Fatal("Wechat.AopF2F.Update.1")
				log.Fatal(err, req)
				return fmt.Errorf("订单状态更新失败:%s", err)
			}
			res.Valid = false
			return err
		}
		e, _ := data.Json() //无法正常返回时
		log.Fatal("Wechat.AopF2F.data")
		log.Fatal(e, req)
		return fmt.Errorf(string(e))
	}
	return err
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
	if err != nil {
		return config, err
	}
	order.StoreId = config.Id //修复商家用户名支付时无法获取商家id问题
	return config, err
}

// GetOrder 获取订单
func (srv *Pay) GetOrder(order *pd.Order) (err error) {
	srv.Order = &orderPB.Order{
		StoreId: order.StoreId, // 商户门店编号 收款账号ID userID
		OrderNo: order.OrderNo, // 订单编号
	}
	return srv.Repo.StoreIdAndOrderNoGet(srv.Order)
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
	if err == gorm.ErrRecordNotFound {
		log.Fatal("StoreIdAndOrderNoGet")
		log.Fatal(err, order)
		err = srv.Repo.Create(srv.Order)
	}
	if srv.Order.StoreId != order.StoreId || srv.Order.OrderNo != order.OrderNo || srv.Order.Method != order.Method || srv.Order.AuthCode != order.AuthCode || srv.Order.TotalAmount != order.TotalAmount {
		log.Fatal("StoreIdAndOrderNoGet=")
		log.Fatal(srv.Order, order)
		return errors.New("上报订单已存在,但数据校验失败")
	}
	return err
}

// newAlipayClient 实例化支付宝付款方式连接
func (srv *Pay) newAlipayClient(c *configPB.Config) {
	srv.Alipay.NewClient(map[string]string{
		"AppId":                c.Alipay.AppId,
		"PrivateKey":           c.Alipay.PrivateKey,
		"AliPayPublicKey":      c.Alipay.AliPayPublicKey,
		"AppAuthToken":         c.Alipay.AppAuthToken,
		"SysServiceProviderId": c.Alipay.SysServiceProviderId,
		"SignType":             c.Alipay.SignType,
	}, c.Alipay.Sandbox)
}

// newWechatClient 实例化微信付款方式连接
func (srv *Pay) newWechatClient(c *configPB.Config) {
	srv.Wechat.NewClient(map[string]string{
		"AppId":    c.Wechat.AppId,
		"MchId":    c.Wechat.MchId,
		"ApiKey":   c.Wechat.ApiKey,
		"SubAppId": c.Wechat.SubAppId,
		"SubMchId": c.Wechat.SubMchId,
	}, c.Wechat.Sandbox)
}
