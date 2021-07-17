package handler

import (
	"context"
	"errors"
	"math"
	"regexp"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/lecex/core/env"
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
	CLOSED     = "CLOSED"     // -1
	USERPAYING = "USERPAYING" // 0
	SUCCESS    = "SUCCESS"    // 1
)

// Pay 支付结构
type Pay struct {
	Config repository.Config
	Repo   repository.Order
	Alipay *service.Alipay
	Wechat *service.Wechat
	Icbc   *service.Icbc
}

// AopF2F 商家扫用户付款码
// https://pay.weixin.qq.com/wiki/doc/api/micropay.php?chapter=9_10&index=1
func (srv *Pay) AopF2F(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	if req.Order.AuthCode == "" {
		return errors.New("付款码不允许为空")
	}
	if req.Order.Title == "" {
		return errors.New("订单标题不允许为空")
	}
	if req.Order.TotalAmount <= 0 {
		return errors.New("订单金额不允许小于1")
	}
	if req.Order.OrderNo == "" {
		return errors.New("订单编号不允许为空")
	}
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
	repoOrder, err := srv.handerOrder(&orderPB.Order{
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
	if repoOrder.Method != req.Order.Method {
		res.Error.Code = "AopF2F.handerOrder.Method"
		res.Error.Detail = "请求支付通道方式和系统已存在订单不符"
		log.Fatal(req, res)
		return nil
	}
	res.Order = req.Order
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
		if content["code"] == "10000" && content["msg"] == "Success" { // 返回状态码
			res.Valid = true
			res.Order.Stauts = SUCCESS
			err = srv.successOrder(repoOrder, config.Icbc.Fee)
			if err != nil {
				res.Order.Stauts = USERPAYING
				res.Error.Code = "Query.Wechat.Update.Success"
				res.Error.Detail = "支付成功,更新订单状态失败!"
				log.Fatal(req, res, err)
			}
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
		if content["return_code"] == "SUCCESS" { // 返回状态码
			if content["result_code"] == "SUCCESS" { // 业务结果
				res.Valid = true
				res.Order.Stauts = SUCCESS
				err = srv.successOrder(repoOrder, config.Icbc.Fee)
				if err != nil {
					res.Order.Stauts = USERPAYING
					res.Error.Code = "Query.Wechat.Update.Success"
					res.Error.Detail = "支付成功,更新订单状态失败!"
					log.Fatal(req, res, err)
				}
			}
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
	case "icbc":
		srv.newIcbcClient(config) //实例化微信连接
		content, err := srv.Icbc.AopF2F(req.Order)
		if err != nil {
			res.Error.Code = "AopF2F.Icbc"
			res.Error.Detail = "工行下单失败:" + err.Error()
			log.Fatal(req, res, err)
			return nil
		}
		if content["return_code"] == "0" { // 返回码
			// 	-1:下单失败，0：支付中，1：支付成功，2：支付失败
			if content["pay_status"] == "1" { // 交易状态标识
				res.Valid = true
				res.Order.Stauts = SUCCESS
				err = srv.successOrder(repoOrder, config.Icbc.Fee)
				if err != nil {
					res.Order.Stauts = USERPAYING
					res.Error.Code = "Query.Icbc.Update.Success"
					res.Error.Detail = "支付成功,更新订单状态失败!"
					log.Fatal(req, res, err)
				}
			}
		}
		c, err := content.Json()
		if err != nil {
			res.Error.Code = "AopF2F.Icbc.Mxj"
			res.Error.Detail = "工行下单返回数据解析失败"
			log.Fatal(req, res, err)
			return nil
		}
		res.Content = string(c) //数据正常返回
		log.Fatal("AopF2F.Icbc", req, res, err)
		return nil
	}
	return nil
}

// Query 支付查询
// https://pay.weixin.qq.com/wiki/doc/api/micropay.php?chapter=9_2
func (srv *Pay) Query(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	if req.Order.OrderNo == "" {
		res.Error.Code = "Query.OrderNo.Empty"
		res.Error.Detail = "查询订单不允许为空"
		log.Fatal(req, res, err)
		return nil
	}
	res.Error = &pd.Error{}
	config, err := srv.userConfig(req.Order)
	if err != nil {
		res.Error.Code = "Query.userConfig"
		res.Error.Detail = "查询配置信息失败"
		log.Fatal(req, res, err)
		return nil
	}
	res.Order = req.Order
	repoOrder, err := srv.getOrder(req.Order) //创建订单返回订单ID
	switch repoOrder.Stauts {
	case -1:
		res.Order.Stauts = CLOSED // 订单状态默认待付款
	case 0:
		res.Order.Stauts = USERPAYING // 订单状态默认待付款
	case 1:
		res.Order.Stauts = SUCCESS // 订单状态默认待付款
	}
	res.Order.StoreId = repoOrder.StoreId
	res.Order.Method = repoOrder.Method
	res.Order.AuthCode = repoOrder.AuthCode
	res.Order.Title = repoOrder.Title
	res.Order.TotalAmount = repoOrder.TotalAmount
	res.Order.OrderNo = repoOrder.OrderNo
	res.Order.OperatorId = repoOrder.OperatorId
	res.Order.TerminalId = repoOrder.TerminalId
	res.Order.RefundFee = repoOrder.RefundFee
	if repoOrder.RefundFee == repoOrder.TotalAmount {
		res.Error.Code = "Query.RefundFee"
		res.Error.Detail = "订单已退款不支持再次查询"
		log.Fatal(req, res, err)
		return nil
	}
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
		return nil
	case "icbc":
		srv.newIcbcClient(config) //实例化连工行接
		content, err := srv.Icbc.Query(req.Order)
		if err != nil {
			res.Error.Code = "Query.Icbc.Error"
			res.Error.Detail = err.Error()
			log.Fatal(req, res, err)
			return nil
		}
		c, err := content.Json()
		if err != nil {
			res.Error.Code = "Query.Icbc.Mxj"
			res.Error.Detail = "工行返回数据解析失败"
			log.Fatal(req, res, err)
			return nil
		}
		res.Content = string(c) //数据正常返回
		// 错误处理
		if content["return_code"] == "0" { // 通信标识
			if content["pay_status"] == "1" { // 交易状态标识
				res.Valid = true
				res.Order.Stauts = SUCCESS
				err = srv.successOrder(repoOrder, config.Icbc.Fee)
				if err != nil {
					res.Order.Stauts = USERPAYING
					res.Error.Code = "Query.Icbc.Update.Success"
					res.Error.Detail = "支付成功,更新订单状态失败!"
					log.Fatal(req, res, err)
				}
			}
			// 交易结果标志：0：支付中请稍后查询，1：支付成功，2：支付失败，3：已撤销，4：撤销中请稍后查询，5：已全额退款，6：已部分退款，7：退款中请稍后查询
			if content["pay_status"] == "-1" || content["pay_status"] == "2" || content["pay_status"] == "3" || content["pay_status"] == "5" {
				repoOrder.Fee = 0
				repoOrder.Stauts = -1
				res.Order.Stauts = CLOSED
				err = srv.Repo.Update(repoOrder)
				if err != nil {
					res.Order.Stauts = USERPAYING
					res.Error.Code = "Query.Icbc.Update.Close"
					res.Error.Detail = "支付失败,更新订单状态失败!"
					log.Fatal(req, res, err)
				}
			}
		} else {
			res.Error.Code = content["return_code"].(string)
			res.Error.Detail = content["return_msg"].(string)
		}
		return nil
	}
	return nil
}

// RefundQuery 退款查询
func (srv *Pay) RefundQuery(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	return errors.New("暂未实现")
}

// Refund 交易退款
func (srv *Pay) Refund(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	res.Error = &pd.Error{}
	config, err := srv.userConfig(req.Order) // 获取配置同时根据商家名称赋值商家id
	req.Order.Method = ""                    // 退款取消默认通道以数据库订单为准
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
	if req.Order.RefundFee >= (originalOrder.TotalAmount - originalOrder.RefundFee) {
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
		req.Order.OrderNo = originalOrder.OrderNo + "_" + time.Now().Format("0102150405") // 全额退款编号自动构建
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
		srv.newWechatClient(config) //实例化连微信接
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
	case "icbc":
		srv.newIcbcClient(config) //实例化连工行接
		content, err := srv.Icbc.Refund(refundOrder, originalOrder)
		if err != nil {
			return valid, res, err
		}
		if content["return_code"] == "0" { // 订单关闭状态
			err = srv.successOrder(refundOrder, config.Icbc.Fee)
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
	if order.StoreId == "" && order.StoreName == "" {
		return nil, errors.New("StoreName,StoreId 必须包含一个")
	}
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
	if order.Method == "" {
		// 默认通道
		if config.Channel != "" && config.Channel != "wechat" && config.Channel != "alipay" {
			order.Method = config.Channel
		} else {
			// 直连通道
			if ok, _ := regexp.Match(`^(?:2[5-9]|30)\d{14,18}$`, []byte(order.AuthCode)); ok {
				order.Method = "alipay"
			}
			if ok, _ := regexp.Match(`^1[0-5]\d{16}$`, []byte(order.AuthCode)); ok {
				order.Method = "wechat"
			}
		}
	}
	order.StoreId = config.Id             //修复商家用户名支付时无法获取商家id问题
	if config.Alipay.AppAuthToken != "" { // 子商户模式需要通过系统配置进行设置服务商信息
		config.Alipay.AppId = env.Getenv("PAY_ALIPAY_APPID", config.Alipay.AppId)
		config.Alipay.PrivateKey = env.Getenv("PAY_ALIPAY_PRIVATE_KEY", config.Alipay.PrivateKey)
		config.Alipay.AliPayPublicKey = env.Getenv("PAY_ALIPAY_ALIPAY_PUBLIC_KEY", config.Alipay.AliPayPublicKey)
		config.Alipay.SignType = env.Getenv("PAY_ALIPAY_SIGN_TYPE", config.Alipay.SignType)
		config.Alipay.SysServiceProviderId = env.Getenv("PAY_ALIPAY_SYS_SERVICE_PROVIDERID", config.Alipay.SysServiceProviderId)
	}
	if config.Wechat.SubMchId != "" { // 子商户模式需要通过系统配置进行设置服务商信息
		config.Wechat.AppId = env.Getenv("PAY_WECHAT_APPID", config.Wechat.AppId)
		config.Wechat.MchId = env.Getenv("PAY_WECHAT_MCHID", config.Wechat.MchId)
		config.Wechat.ApiKey = env.Getenv("PAY_WECHAT_APIKEY", config.Wechat.ApiKey)
		config.Wechat.PemCert = env.Getenv("PAY_WECHAT_PEMCERT", config.Wechat.PemCert)
		config.Wechat.PemKey = env.Getenv("PAY_WECHAT_PEMKEY", config.Wechat.PemKey)
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

// newIcbcClient 实例化微信付款方式连接
func (srv *Pay) newIcbcClient(c *configPB.Config) {
	srv.Icbc.NewClient(map[string]string{
		"AppId":          c.Icbc.AppId,
		"PrivateKey":     c.Icbc.PrivateKey,
		"IcbcPublicKey":  c.Icbc.IcbcPublicKey,
		"SignType":       c.Icbc.SignType,
		"ReturnSignType": c.Icbc.ReturnSignType,
		"MerId":          c.Icbc.MerId,
	})
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
