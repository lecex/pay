package handler

import (
	"context"
	"errors"
	"math"

	"github.com/jinzhu/gorm"
	configPB "github.com/lecex/pay/proto/config"
	orderPB "github.com/lecex/pay/proto/order"
	pd "github.com/lecex/pay/proto/pay"
	"github.com/lecex/pay/service"
	"github.com/lecex/pay/service/repository"
	"github.com/micro/go-micro/v2/util/log"
)

// USERPAYING 代付款
// SUCCESS 付款成功
// CLOSED 订单关闭
const (
	USERPAYING = "USERPAYING"
	SUCCESS    = "SUCCESS"
	CLOSED     = "CLOSED"
)

// Pay 支付结构
type Pay struct {
	Config repository.Config
	Repo   repository.Order
	Alipay *service.Alipay
	Wechat *service.Wechat
}

// Query 支付查询
// https://pay.weixin.qq.com/wiki/doc/api/micropay.php?chapter=9_2
func (srv *Pay) Query(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	res.Error = &pd.Error{}
	res.Order = req.Order
	res.Order.Stauts = USERPAYING // 订单状态默认代付款
	config, err := srv.UserConfig(req.Order)
	if err != nil {
		res.Error.Code = "Query.UserConfig"
		res.Error.Detail = "查询配置信息失败"
		log.Fatal(req, res, err)
		return nil
	}
	repoOrder, err := srv.GetOrder(req.Order) //创建订单返回订单ID
	if err != nil {
		res.Order.Stauts = CLOSED
		res.Error.Code = "Query.GetOrder"
		res.Error.Detail = "获取订单失败"
		log.Fatal(req, res, err)
		return nil
	}
	// if repoOrder.Stauts == 1 {
	// 	res.Valid = true
	// 	return err // 支付成功返回
	// }
	// if repoOrder.Stauts == -1 {
	// 	return fmt.Errorf("订单已关闭")
	// }
	switch repoOrder.Method {
	case "alipay":
		srv.newAlipayClient(config) //实例化支付宝连接
		content, err := srv.Alipay.Query(req.Order)
		if err != nil {
			res.Error.Code = "Query.Alipay.Error"
			res.Error.Detail = err.Error()
			log.Fatal(req, res, err)
			return nil
		}
		c, err := content.Json()
		if err != nil {
			res.Error.Code = "Query.Alipay.Mxj"
			res.Error.Detail = "支付宝返回数据解析失败"
			log.Fatal(req, res, err)
			return nil
		}
		res.Content = string(c) //数据正常返回
		log.Fatal("Query.Alipay", req, res, err)
		if (content["trade_status"] == "TRADE_SUCCESS" || content["trade_status"] == "TRADE_FINISHED") && content["code"] == "10000" && content["msg"] == "Success" { // 订单成功状态
			res.Valid = true
			res.Order.Stauts = SUCCESS
			err = srv.successOrder(repoOrder, config.Alipay.Fee)
			if err != nil {
				res.Error.Code = "Query.Alipay.Update.Success"
				res.Error.Detail = "支付成功,更新订单状态失败!"
				log.Fatal(req, res, err)
			}
			return nil
		}
		if content["trade_status"] == "TRADE_CLOSED" || content["sub_code"] == "ACQ.TRADE_NOT_EXIST" { // 订单关闭状态
			repoOrder.Fee = 0
			repoOrder.Stauts = -1
			res.Order.Stauts = CLOSED
			err = srv.Repo.Update(repoOrder)
			if err != nil {
				res.Error.Code = "Query.Alipay.Update.Close"
				res.Error.Detail = "支付失败,更新订单状态失败!"
				log.Fatal(req, res, err)
			}
		}
		if content["sub_code"] != nil { // 返回错误代码
			res.Error.Code = content["sub_code"].(string)
			res.Error.Detail = content["sub_msg"].(string)
		}
		return nil
	case "wechat":
		srv.newWechatClient(config) //实例化连微信接
		content, err := srv.Wechat.Query(req.Order)
		if err != nil {
			res.Error.Code = "Query.Wechat.Error"
			res.Error.Detail = err.Error()
			log.Fatal(req, res, err)
			return nil
		}
		c, err := content.Json()
		if err != nil {
			res.Error.Code = "Query.Wechat.Mxj"
			res.Error.Detail = "微信返回数据解析失败"
			log.Fatal(req, res, err)
			return nil
		}
		res.Content = string(c) //数据正常返回
		log.Fatal("Query.Wechat", req, res, err)
		// 错误处理
		if content["return_code"] == "SUCCESS" { // 通信标识
			if content["trade_state"] == "SUCCESS" { // 交易状态标识
				res.Valid = true
				res.Order.Stauts = SUCCESS
				err = srv.successOrder(repoOrder, config.Wechat.Fee)
				if err != nil {
					res.Order.Stauts = USERPAYING
					res.Error.Code = "Query.Wechat.Update.Success"
					res.Error.Detail = "支付成功,更新订单状态失败!"
					log.Fatal(req, res, err)
				}
				return nil
			}
			// SUCCESS—支付成功、REFUND—转入退款、NOTPAY—未支付、CLOSED—已关闭、REVOKED—已撤销（付款码支付）、USERPAYING--用户支付中（付款码支付）、PAYERROR--支付失败(其他原因，如银行返回失败)
			if content["trade_state"] == "REFUND" || content["trade_state"] == "CLOSED" || content["trade_state"] == "REVOKED" || content["trade_state"] == "PAYERROR" || content["err_code"] == "ORDERNOTEXIST" {
				repoOrder.Fee = 0
				repoOrder.Stauts = -1
				res.Order.Stauts = CLOSED
				err = srv.Repo.Update(repoOrder)
				if err != nil {
					res.Order.Stauts = USERPAYING
					res.Error.Code = "Query.Wechat.Update.Close"
					res.Error.Detail = "支付失败,更新订单状态失败!"
					log.Fatal(req, res, err)
				}
			}
			if content["result_code"] != "SUCCESS" { // 返回错误代码
				res.Error.Code = content["err_code"].(string)
				res.Error.Detail = content["err_code_des"].(string)
			}
		} else {
			res.Error.Code = "Query.Wechat.ReturnCode"
			res.Error.Detail = content["return_msg"].(string)
		}
		return nil
	}
	return nil
}

// AopF2F 商家扫用户付款码
// https://pay.weixin.qq.com/wiki/doc/api/micropay.php?chapter=9_10&index=1
func (srv *Pay) AopF2F(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	res.Error = &pd.Error{}
	config, err := srv.UserConfig(req.Order)
	if err != nil {
		res.Error.Code = "AopF2F.UserConfig"
		res.Error.Detail = "查询配置信息失败"
		log.Fatal(req, res, err)
		return nil
	}
	if !config.Stauts {
		res.Error.Code = "AopF2F.Stauts"
		res.Error.Detail = "支付功能被禁用！请联系管理员。"
		return nil
	}
	_, err = srv.HanderOrder(req.Order) //创建订单返回订单ID
	if err != nil {
		res.Error.Code = "AopF2F.HanderOrder"
		res.Error.Detail = "创建订单失败"
		log.Fatal(req, res, err)
		return nil
	}
	// if repoOrder.Stauts == 1 {
	// 	res.Valid = true
	// 	return err // 支付成功返回
	// }
	// if repoOrder.Stauts == -1 {
	// 	return fmt.Errorf("订单已关闭")
	// }
	switch req.Order.Method {
	case "alipay":
		srv.newAlipayClient(config) //实例化连支付宝接
		content, err := srv.Alipay.AopF2F(req.Order)
		if err != nil {
			res.Error.Code = "AopF2F.Alipay"
			res.Error.Detail = "支付宝下单失败:" + err.Error()
			log.Fatal(req, res, err)
			return nil
		}
		c, err := content.Json()
		if err != nil {
			res.Error.Code = "AopF2F.Alipay.Mxj"
			res.Error.Detail = "支付宝下单返回数据解析失败"
			log.Fatal(req, res, err)
			return nil
		}
		res.Content = string(c) //数据正常返回
		log.Fatal("AopF2F.AopF2F", req, res, err)
		return nil
	case "wechat":
		srv.newWechatClient(config) //实例化微信连接
		content, err := srv.Wechat.AopF2F(req.Order)
		if err != nil {
			res.Error.Code = "AopF2F.Wechat"
			res.Error.Detail = "微信下单失败:" + err.Error()
			log.Fatal(req, res, err)
			return nil
		}
		c, err := content.Json()
		if err != nil {
			res.Error.Code = "AopF2F.Wechat.Mxj"
			res.Error.Detail = "微信下单返回数据解析失败"
			log.Fatal(req, res, err)
			return nil
		}
		res.Content = string(c) //数据正常返回
		log.Fatal("AopF2F.Wechat", req, res, err)
		return nil
	}
	return nil
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
func (srv *Pay) GetOrder(order *pd.Order) (repoOrder *orderPB.Order, err error) {
	repoOrder = &orderPB.Order{
		StoreId: order.StoreId, // 商户门店编号 收款账号ID userID
		OrderNo: order.OrderNo, // 订单编号
	}
	err = srv.Repo.StoreIdAndOrderNoGet(repoOrder)
	return repoOrder, err
}

// HanderOrder 处理订单
func (srv *Pay) HanderOrder(order *pd.Order) (repoOrder *orderPB.Order, err error) {
	repoOrder = &orderPB.Order{
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
	err = srv.Repo.StoreIdAndOrderNoGet(repoOrder)
	if err == gorm.ErrRecordNotFound {
		err = srv.Repo.Create(repoOrder)
		if err != nil {
			log.Fatal("Order.Create")
			log.Fatal(err, order)
			return nil, err
		}
	}
	if repoOrder.StoreId != order.StoreId || repoOrder.OrderNo != order.OrderNo || repoOrder.Method != order.Method || repoOrder.AuthCode != order.AuthCode || repoOrder.TotalAmount != order.TotalAmount {
		log.Fatal("OrderVsOrder")
		log.Fatal(repoOrder, order)
		return nil, errors.New("上报订单已存在,但数据校验失败")
	}
	return repoOrder, err
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

// 支付成功订单处理
func (srv *Pay) successOrder(repoOrder *orderPB.Order, fee int64) (err error) {
	repoOrder.Stauts = 1
	repoOrder.Fee = int64(math.Floor(float64(repoOrder.TotalAmount*fee)/10000 + 0.5)) // 相乘后转浮点型乘以万分之一然后四舍五入 【+0.5四舍五入取整】
	err = srv.Repo.Update(repoOrder)
	if err != nil {
		return err
	}
	return err
}
