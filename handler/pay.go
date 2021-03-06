package handler

import (
	"context"
	"math"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/lecex/core/env"
	"github.com/lecex/core/uitl"
	configPB "github.com/lecex/pay/proto/config"
	orderPB "github.com/lecex/pay/proto/order"
	pd "github.com/lecex/pay/proto/pay"
	"github.com/lecex/pay/service"
	"github.com/lecex/pay/service/repository"
	"github.com/micro/go-micro/v2/util/log"
)

// USERPAYING 待付款|待退款
// SUCCESS 付款成功
// CLOSED 订单关闭
const (
	USERPAYING = "USERPAYING" // 0
	SUCCESS    = "SUCCESS"    // 1
	CLOSED     = "CLOSED"     // -1
)

// Pay 支付结构
type Pay struct {
	Config repository.Config
	Repo   repository.Order
	Alipay *service.Alipay
	Wechat *service.Wechat
}

// AopF2F 商家扫用户付款码
// https://pay.weixin.qq.com/wiki/doc/api/micropay.php?chapter=9_10&index=1
func (srv *Pay) AopF2F(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	res.Error = &pd.Error{}
	config, err := srv.userConfig(req.Order)
	if err != nil {
		res.Error.Code = "AopF2F.userConfig"
		res.Error.Detail = "查询配置信息失败"
		log.Fatal(req, res, err)
		return nil
	}
	if !config.Stauts {
		res.Error.Code = "AopF2F.Stauts"
		res.Error.Detail = "支付功能被禁用！请联系管理员。"
		return nil
	}
	_, err = srv.handerOrder(&orderPB.Order{
		StoreId:     req.Order.StoreId,     // 商户门店编号 收款账号ID userID
		Method:      req.Order.Method,      // 付款方式 [支付宝、微信、银联等]
		AuthCode:    req.Order.AuthCode,    // 付款码
		Title:       req.Order.Title,       // 订单标题
		TotalAmount: req.Order.TotalAmount, // 订单总金额
		OrderNo:     req.Order.OrderNo,     // 订单编号
		OperatorId:  req.Order.OperatorId,  // 商户操作员编号
		TerminalId:  req.Order.TerminalId,  // 商户机具终端编号
		Stauts:      0,                     // 订单状态 默认状态未付款
	}) //创建订单返回订单ID
	if err != nil {
		res.Error.Code = "AopF2F.handerOrder"
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

// Query 支付查询
// https://pay.weixin.qq.com/wiki/doc/api/micropay.php?chapter=9_2
func (srv *Pay) Query(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	res.Error = &pd.Error{}
	res.Order = req.Order
	res.Order.Stauts = USERPAYING // 订单状态默认待付款
	config, err := srv.userConfig(req.Order)
	if err != nil {
		res.Error.Code = "Query.userConfig"
		res.Error.Detail = "查询配置信息失败"
		log.Fatal(req, res, err)
		return nil
	}
	repoOrder, err := srv.getOrder(req.Order) //创建订单返回订单ID
	if err != nil {
		res.Order.Stauts = CLOSED
		res.Error.Code = "Query.GetOrder"
		res.Error.Detail = "获取订单失败"
		log.Fatal(req, res, err)
		return nil
	}
	if repoOrder.TotalAmount < 0 { // 退款查询时不进行是不进行实际查询等待系统自动结果
		res.Order.Method = repoOrder.Method
		switch repoOrder.Stauts {
		case -1:
			res.Order.Stauts = CLOSED
		case 0:
			res.Order.Stauts = USERPAYING
		case 1:
			res.Order.Stauts = SUCCESS
		}
		return nil
	}
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
		res.Content = string(c)                                                                                                                                       //数据正常返回
		if (content["trade_status"] == "TRADE_SUCCESS" || content["trade_status"] == "TRADE_FINISHED") && content["code"] == "10000" && content["msg"] == "Success" { // 订单成功状态
			res.Valid = true
			res.Order.Stauts = SUCCESS
			err = srv.successOrder(repoOrder, config.Alipay.Fee)
			if err != nil {
				res.Error.Code = "Query.Alipay.Update.Success"
				res.Error.Detail = "支付成功,更新订单状态失败!"
				log.Fatal(req, res, err)
			}
		}
		if content["trade_status"] == "TRADE_CLOSED" || content["sub_code"] == "ACQ.TRADE_NOT_EXIST" { // 订单关闭状态
			if repoOrder.RefundFee == 0 { // 不存在退款时才可以关闭订单
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
		}
		if content["sub_code"] != nil { // 返回错误代码
			res.Error.Code = content["sub_code"].(string)
			res.Error.Detail = content["sub_msg"].(string)
		}
		log.Fatal("Query.Alipay", req, res, err)
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
		log.Fatal("Query.Wechat", req, res, err)
		return nil
	}
	return nil
}

// Cancel 交易撤销
func (srv *Pay) Cancel(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	res.Error = &pd.Error{}
	res.Order = req.Order
	config, err := srv.userConfig(req.Order)
	if err != nil {
		res.Error.Code = "Cancel.userConfig"
		res.Error.Detail = "撤销查询配置信息失败"
		log.Fatal(req, res, err)
		return nil
	}
	repoOrder, err := srv.getOrder(req.Order) //获取订单
	if err != nil {
		res.Error.Code = "Cancel.getOrder"
		res.Error.Detail = "撤销获取订单失败"
		log.Fatal(req, res, err)
		return nil
	}
	createdAt, err := time.ParseInLocation("2006-01-02T15:04:05+08:00", repoOrder.CreatedAt, time.Local)
	if err != nil {
		res.Error.Code = "Cancel.time.ParseInLocation"
		res.Error.Detail = "撤销订单获取订单创建时间失败"
		log.Fatal(req, res, err)
		return nil
	}
	if uitl.GetZeroTime(createdAt) != uitl.GetZeroTime(time.Now()) {
		res.Error.Code = "Cancel.createdAt.Not.SameDay"
		res.Error.Detail = "只能撤销当天订单"
		log.Fatal(req, res, err)
		return nil
	}
	switch repoOrder.Method {
	case "alipay":
		srv.newAlipayClient(config) //实例化连支付宝接
		content, err := srv.Alipay.Cancel(req.Order)
		if err != nil {
			res.Error.Code = "Cancel.Alipay"
			res.Error.Detail = "支付宝下单失败:" + err.Error()
			log.Fatal(req, res, err)
			return nil
		}
		if content["code"] == "10000" && content["msg"] == "Success" { // 订单关闭状态
			res.Valid = true
			repoOrder.Fee = 0 // 手续费改为 0
			repoOrder.Stauts = -1
			res.Order.Stauts = CLOSED
			err = srv.Repo.Update(repoOrder)
			if err != nil {
				res.Error.Code = "Cancel.Alipay.Update.Close"
				res.Error.Detail = "支付宝订单撤销,更新订单状态失败!"
				log.Fatal(req, res, err)
			}
		}
		c, err := content.Json()
		if err != nil {
			res.Error.Code = "Cancel.Alipay.Mxj"
			res.Error.Detail = "支付宝订单撤销返回数据解析失败"
			log.Fatal(req, res, err)
			return nil
		}
		res.Content = string(c) //数据正常返回
		log.Fatal("Cancel", req, res, err)
		return nil
	case "wechat":
		srv.newWechatClient(config) //实例化连支付宝接
		content, err := srv.Wechat.Cancel(req.Order)
		if err != nil {
			res.Error.Code = "Refund.Wechat"
			res.Error.Detail = "微信撤销失败:" + err.Error()
			log.Fatal(req, res, err)
			return nil
		}
		if content["return_code"] == "SUCCESS" && content["result_code"] == "SUCCESS" { // 订单关闭状态
			res.Valid = true
			repoOrder.Fee = 0 // 手续费改为 0
			repoOrder.Stauts = -1
			res.Order.Stauts = CLOSED
			err = srv.Repo.Update(repoOrder)
			if err != nil {
				res.Error.Code = "Refund.Wechat.Update.Close"
				res.Error.Detail = "微信订单撤销,更新订单状态失败!"
				log.Fatal(req, res, err)
			}
		}
		c, err := content.Json()
		if err != nil {
			res.Error.Code = "Refund.Wechat.Mxj"
			res.Error.Detail = "微信订单撤销返回数据解析失败"
			log.Fatal(req, res, err)
			return nil
		}
		res.Content = string(c) //数据正常返回
		log.Fatal("Refund", req, res, err)
		return nil
	}
	return nil
}

// Refund 交易退款
func (srv *Pay) Refund(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	res.Error = &pd.Error{}
	config, err := srv.userConfig(req.Order) // 获取配置同时根据商家名称赋值商家id
	if err != nil {
		res.Error.Code = "Refund.userConfig"
		res.Error.Detail = "退款是支付配置信息查询失败"
		log.Fatal(req, res, err)
		return nil
	}
	originalOrder := &orderPB.Order{
		StoreId: req.Order.StoreId,         // 商户门店编号 收款账号ID userID
		OrderNo: req.Order.OriginalOrderNo, // 订单编号
	}
	err = srv.Repo.StoreIdAndOrderNoGet(originalOrder) //创建订单返回订单ID
	if err != nil {
		res.Error.Code = "Refund.getOrder"
		res.Error.Detail = "退款获取订单失败"
		log.Fatal(req, res, err)
		return nil
	}
	if originalOrder.Stauts != 1 {
		res.Error.Code = "Refund.Stauts"
		res.Error.Detail = "订单未支付成功不允许退款"
		log.Fatal(req, originalOrder)
		return nil
	}
	if req.Order.RefundFee > (originalOrder.TotalAmount - originalOrder.RefundFee) {
		res.Error.Code = "Refund.RefundFee"
		res.Error.Detail = "退款金额大于可退款金额"
		log.Fatal(req, originalOrder)
		return nil
	}
	// 构建新的退款订单
	if req.Order.RefundFee == 0 {
		req.Order.TotalAmount = -originalOrder.TotalAmount // 全额退款
	} else {
		req.Order.TotalAmount = -req.Order.RefundFee // 退款改为负数金额
	}
	if req.Order.OrderNo == "" {
		req.Order.OrderNo = originalOrder.OrderNo + "_Q" // 全额退款编号自动构建
	}
	refundOrder, err := srv.handerOrder(&orderPB.Order{
		StoreId:     originalOrder.StoreId,    // 商户门店编号 收款账号ID userID
		Method:      originalOrder.Method,     // 付款方式 [支付宝、微信、银联等]
		AuthCode:    originalOrder.AuthCode,   // 付款码
		Title:       originalOrder.Title,      // 订单标题
		TotalAmount: req.Order.TotalAmount,    // 订单总金额
		OrderNo:     req.Order.OrderNo,        // 订单编号
		OperatorId:  originalOrder.OperatorId, // 商户操作员编号
		TerminalId:  originalOrder.TerminalId, // 商户机具终端编号
		LinkId:      originalOrder.Id,
		Stauts:      0, // 订单状态 默认状态未付款
	}) //创建退款订单返回订单ID
	if req.Order.Verify { // 需要验证授权
		res.Order = req.Order
		res.Order.Method = originalOrder.Method
		res.Valid = true
	} else {
		res.Valid, res.Content, err = srv.handerRefund(config, refundOrder, originalOrder)
	}
	return nil
}

// AffirmRefund 确认退款
func (srv *Pay) AffirmRefund(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	res.Error = &pd.Error{}
	config, err := srv.userConfig(req.Order) // 获取配置同时根据商家名称赋值商家id
	if err != nil {
		res.Error.Code = "AffirmRefund.userConfig"
		res.Error.Detail = "退款是支付配置信息查询失败"
		log.Fatal(req, res, err)
		return nil
	}
	refundOrder, err := srv.getOrder(req.Order) //创建订单返回订单ID
	if err != nil {
		res.Error.Code = "AffirmRefund.getOrder"
		res.Error.Detail = "确认退款获取订单失败"
		log.Fatal(req, res, err)
		return nil
	}
	originalOrder := &orderPB.Order{
		Id: refundOrder.LinkId,
	}
	err = srv.Repo.Get(originalOrder)
	if err != nil {
		res.Error.Code = "AffirmRefund.Repo.Get"
		res.Error.Detail = "确认退款获取原始订单失败"
		log.Fatal(req, res, err)
		return nil
	}
	res.Valid, res.Content, err = srv.handerRefund(config, refundOrder, originalOrder)
	return
}

// handerRefund 处理退款
func (srv *Pay) handerRefund(config *configPB.Config, refundOrder *orderPB.Order, originalOrder *orderPB.Order) (valid bool, res string, err error) {
	switch refundOrder.Method {
	case "alipay":
		srv.newAlipayClient(config) //实例化连支付宝接
		content, err := srv.Alipay.Refund(refundOrder, originalOrder)
		if err != nil {
			return valid, res, err
		}
		if content["code"] == "10000" && content["msg"] == "Success" { // 订单关闭状态
			err = srv.successOrder(refundOrder, config.Alipay.Fee)
			if err != nil {
				return valid, res, err
			}
			originalOrder.RefundFee = originalOrder.RefundFee + (-refundOrder.TotalAmount) // 已有退款加正数退款
			err = srv.Repo.Update(originalOrder)
			if err != nil {
				return valid, res, err
			}
			valid = true
		}
		if content["sub_code"] == "ACQ.TRADE_HAS_CLOSE" || content["sub_code"] == "ACQ.TRADE_NOT_EXIST" { // 订单关闭状态
			refundOrder.Fee = 0
			refundOrder.Stauts = -1
			err = srv.Repo.Update(refundOrder)
			if err != nil {
				return valid, res, err
			}
		}
		c, err := content.Json()
		if err != nil {
			return valid, res, err
		}
		res = string(c) //数据正常返回
		return valid, res, err
	case "wechat":
		srv.newWechatClient(config) //实例化连支付宝接
		content, err := srv.Wechat.Refund(refundOrder, originalOrder)
		if err != nil {
			return valid, res, err
		}
		if content["return_code"] == "SUCCESS" && content["result_code"] == "SUCCESS" { // 订单关闭状态
			err = srv.successOrder(refundOrder, config.Wechat.Fee)
			if err != nil {
				return valid, res, err
			}
			originalOrder.RefundFee = originalOrder.RefundFee + (-refundOrder.TotalAmount) // 已有退款加正数退款
			err = srv.Repo.Update(originalOrder)
			if err != nil {
				return valid, res, err
			}
			valid = true
		}
		c, err := content.Json()
		if err != nil {
			return valid, res, err
		}
		res = string(c) //数据正常返回
		return valid, res, err
	}
	return valid, res, err
}

// userConfig 用户配置
func (srv *Pay) userConfig(order *pd.Order) (*configPB.Config, error) {
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
	if config.Alipay.AppAuthToken != "" {
		config.Alipay.AppId = env.Getenv("PAY_ALIPAY_APPID", config.Alipay.AppId)
		config.Alipay.PrivateKey = env.Getenv("PAY_ALIPAY_PRIVATE_KEY", config.Alipay.PrivateKey)
		config.Alipay.AliPayPublicKey = env.Getenv("PAY_ALIPAY_ALIPAY_PUBLIC_KEY", config.Alipay.AliPayPublicKey)
		config.Alipay.SignType = env.Getenv("PAY_ALIPAY_SIGN_TYPE", config.Alipay.SignType)
		config.Alipay.SysServiceProviderId = env.Getenv("PAY_ALIPAY_SYS_SERVICE_PROVIDERID", config.Alipay.SysServiceProviderId)
	}
	if config.Wechat.SubMchId != "" {
		config.Wechat.AppId = env.Getenv("PAY_ALIPAY_APPID", config.Wechat.AppId)
		config.Wechat.MchId = env.Getenv("PAY_ALIPAY_MCHID", config.Wechat.MchId)
		config.Wechat.ApiKey = env.Getenv("PAY_ALIPAY_APIKEY", config.Wechat.ApiKey)
		config.Wechat.PemCert = env.Getenv("PAY_ALIPAY_PEMCERT", config.Wechat.PemCert)
		config.Wechat.PemKey = env.Getenv("PAY_ALIPAY_PEMKEY", config.Wechat.PemKey)
	}
	return config, err
}

// getOrder 获取订单
func (srv *Pay) getOrder(order *pd.Order) (repoOrder *orderPB.Order, err error) {
	repoOrder = &orderPB.Order{
		StoreId: order.StoreId, // 商户门店编号 收款账号ID userID
		OrderNo: order.OrderNo, // 订单编号
	}
	err = srv.Repo.StoreIdAndOrderNoGet(repoOrder)
	return repoOrder, err
}

// handerOrder 处理订单
func (srv *Pay) handerOrder(repoOrder *orderPB.Order) (*orderPB.Order, error) {
	err := srv.Repo.StoreIdAndOrderNoGet(repoOrder)
	if err == gorm.ErrRecordNotFound {
		err = srv.Repo.Create(repoOrder)
		if err != nil {
			log.Fatal("Order.Create")
			log.Fatal(err, repoOrder)
			return nil, err
		}
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
		"PemCert":  c.Wechat.PemCert,
		"PemKey":   c.Wechat.PemKey,
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
