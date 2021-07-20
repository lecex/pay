package handler

import (
	"context"
	"errors"
	"math"
	"regexp"

	"github.com/clbanning/mxj"
	"github.com/jinzhu/gorm"
	"github.com/micro/go-micro/v2/util/log"

	"github.com/lecex/core/env"

	configPB "github.com/lecex/pay/proto/config"
	orderPB "github.com/lecex/pay/proto/order"
	pd "github.com/lecex/pay/proto/trade"
	"github.com/lecex/pay/service/repository"
	"github.com/lecex/pay/service/trade"
)

// USERPAYING 待付款|待退款
// SUCCESS 付款成功
// CLOSED 订单关闭
const (
	CLOSED     = "CLOSED"     // -1
	USERPAYING = "USERPAYING" // 0
	SUCCESS    = "SUCCESS"    // 1
	WAITING    = "WAITING"    // 1
)

// Trade 支付结构
type Trade struct {
	Config repository.Config
	Repo   repository.Order
	Alipay *trade.Alipay
	Wechat *trade.Wechat
	Icbc   *trade.Icbc
	con    *configPB.Config
}

// AopF2F 商家扫用户付款码
// https://pay.weixin.qq.com/wiki/doc/api/micropay.php?chapter=9_10&index=1
func (srv *Trade) AopF2F(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	// 初始化回参
	res.Content = &pd.Content{}
	if req.BizContent.AuthCode == "" {
		res.Content.ReturnCode = "AopF2F.AuthCode.Not"
		res.Content.ReturnMsg = "付款码不允许为空:BizContent.AuthCode"
		return
	}
	if req.BizContent.Title == "" {
		res.Content.ReturnCode = "AopF2F.Title.Not"
		res.Content.ReturnMsg = "订单标题不允许为空:BizContent.Title"
		return
	}
	if req.BizContent.TotalFee <= 0 {
		res.Content.ReturnCode = "AopF2F.TotalFee.Not"
		res.Content.ReturnMsg = "订单金额不允许小于1:BizContent.TotalFee"
		return
	}
	if req.BizContent.OutTradeNo == "" {
		res.Content.ReturnCode = "AopF2F.OutTradeNo.Not"
		res.Content.ReturnMsg = "订单编号不允许为空:BizContent.OutTradeNo"
		return
	}
	err = srv.userConfig(req)
	if err != nil {
		res.Content.ReturnCode = "AopF2F.userConfig"
		res.Content.ReturnMsg = "查询商户支付配置信息失败"
		log.Fatal(req, res, err)
		return nil
	}
	if !srv.con.Status {
		res.Content.ReturnCode = "AopF2F.Status"
		res.Content.ReturnMsg = "支付功能被禁用！请联系管理员。"
		log.Fatal(req, res, err)
		return nil
	}
	repoOrder, err := srv.handerOrder(&orderPB.Order{
		StoreId:    req.StoreId,               // 商户门店编号 收款账号ID userID
		Channel:    req.BizContent.Channel,    // 付款方式 [支付宝、微信、银联等]
		AuthCode:   req.BizContent.AuthCode,   // 付款码
		Title:      req.BizContent.Title,      // 订单标题
		TotalFee:   req.BizContent.TotalFee,   // 订单总金额
		OutTradeNo: req.BizContent.OutTradeNo, // 订单编号
		OperatorId: req.BizContent.OperatorId, // 商户操作员编号
		TerminalId: req.BizContent.TerminalId, // 商户机具终端编号
		Status:     0,                         // 订单状态 默认状态未付款
		Attach:     req.BizContent.Attach,     // 商户机具终端编号
	}) //创建订单返回订单ID
	if err != nil {
		res.Content.ReturnCode = "AopF2F.handerOrder"
		res.Content.ReturnMsg = "创建系统订单失败"
		log.Fatal(req, res, err)
		return nil
	}
	if repoOrder.Channel != req.BizContent.Channel {
		res.Content.ReturnCode = "AopF2F.handerOrder.BizContent.Channel"
		res.Content.ReturnMsg = "请求支付通道方式和系统已存在订单支付通道不符"
		log.Fatal(req, res)
		return nil
	}
	content := mxj.New()
	switch req.BizContent.Channel {
	case "alipay":
		srv.newAlipayClient() //实例化连支付宝连接
		content, err = srv.Alipay.AopF2F(req.BizContent)
	case "wechat":
		srv.newWechatClient() //实例化微信连接
		content, err = srv.Wechat.AopF2F(req.BizContent)
	case "icbc":
		srv.newIcbcClient() //实例化微信连接
		content, err = srv.Icbc.AopF2F(req.BizContent)
	}
	if err != nil {
		res.Content.ReturnCode = "AopF2F." + req.BizContent.Channel + ".Error"
		res.Content.ReturnMsg = req.BizContent.Channel + "下单请求失败:" + err.Error()
		log.Fatal(res, err)
		return nil
	}
	srv.handerAopF2F(content, res, repoOrder)
	return err
}

// // Query 支付查询
// // https://pay.weixin.qq.com/wiki/doc/api/micropay.php?chapter=9_2
func (srv *Trade) Query(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	// 初始化回参
	res.Content = &pd.Content{}
	if req.BizContent.OutTradeNo == "" {
		res.Content.ReturnCode = "Query.OutTradeNo.Not"
		res.Content.ReturnMsg = "订单编号不允许为空:BizContent.OutTradeNo"
		return
	}
	err = srv.userConfig(req)
	if err != nil {
		res.Content.ReturnCode = "Query.userConfig"
		res.Content.ReturnMsg = "查询商户支付配置信息失败"
		log.Fatal(req, res, err)
		return nil
	}
	if !srv.con.Status {
		res.Content.ReturnCode = "Query.Status"
		res.Content.ReturnMsg = "支付功能被禁用！请联系管理员。"
		log.Fatal(req, res, err)
		return nil
	}
	repoOrder, err := srv.getOrder(req) //
	if err != nil {
		res.Content.ReturnCode = "Query.GetOrder"
		res.Content.ReturnMsg = "获取订单失败"
		log.Fatal(req, res, err)
		return nil
	}

	if repoOrder.TotalFee < 0 { // 退款查询时不进行是不进行实际查询等待系统自动结果
		res.Content.ReturnCode = "Query.Return.Order"
		res.Content.ReturnMsg = "退款订单请使用退款查询接口查询订单"
		log.Fatal(req, res, err)
		return nil
	}

	// debug 退款订单是否允许再次查询
	// if repoOrder.RefundFee == repoOrder.TotalFee {
	// 	res.Content.ReturnCode = "Query.RefundFee.Not.TotalFee"
	// 	res.Content.ReturnMsg = "订单已退款不支持再次查询"
	// 	log.Fatal(req, res, err)
	// 	return nil
	// }

	content := mxj.New()
	switch repoOrder.Channel {
	case "alipay":
		srv.newAlipayClient() //实例化支付宝连接
		content, err = srv.Alipay.Query(req.BizContent)
	case "wechat":
		srv.newWechatClient() //实例化微信连接
		content, err = srv.Wechat.Query(req.BizContent)
	case "icbc":
		srv.newIcbcClient() //实例化微信连接
		content, err = srv.Icbc.Query(req.BizContent)
	}
	if err != nil {
		res.Content.ReturnCode = "Query." + req.BizContent.Channel + ".Error"
		res.Content.ReturnMsg = req.BizContent.Channel + "查询请求失败:" + err.Error()
		log.Fatal(res, err)
		return nil
	}
	srv.handerQuery(content, res, repoOrder)
	return nil
}

// Refund 交易退款
func (srv *Trade) Refund(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	// 初始化回参
	res.Content = &pd.Content{}
	if req.BizContent.OutRefundNo == "" {
		res.Content.ReturnCode = "Refund.OutRefundNo.Not"
		res.Content.ReturnMsg = "退款订单号不允许为空:BizContent.OutRefundNo"
		return
	}
	if req.BizContent.OutTradeNo == "" {
		res.Content.ReturnCode = "Refund.OutTradeNo.Not"
		res.Content.ReturnMsg = "订单编号不允许为空:BizContent.OutTradeNo"
		return
	}

	err = srv.userConfig(req)
	if err != nil {
		res.Content.ReturnCode = "Refund.userConfig"
		res.Content.ReturnMsg = "查询商户支付配置信息失败"
		log.Fatal(req, res, err)
		return nil
	}
	if !srv.con.Status {
		res.Content.ReturnCode = "Refund.Status"
		res.Content.ReturnMsg = "支付功能被禁用！请联系管理员。"
		log.Fatal(req, res, err)
		return nil
	}
	originalOrder := &orderPB.Order{
		StoreId:    req.StoreId,               // 商户门店编号 收款账号ID userID
		OutTradeNo: req.BizContent.OutTradeNo, // 订单编号
	}
	err = srv.Repo.StoreIdAndOutTradeNoGet(originalOrder) //获取订单返回订单ID
	if err != nil {
		res.Content.ReturnCode = "Refund.OriginalOrder.GetOrder"
		res.Content.ReturnMsg = "获取订单失败,请先查询订单确认订单存在"
		log.Fatal(req, res, err)
		return nil
	}
	if originalOrder.Status != 1 {
		res.Content.ReturnCode = "Refund.OriginalOrder.Status"
		res.Content.ReturnMsg = "订单未支付成功不允许退款,请先查询订单确认订单支付成功"
		log.Fatal(req, originalOrder)
		return nil
	}
	// 构建新的退款订单
	if req.BizContent.RefundFee == 0 {
		req.BizContent.TotalFee = -originalOrder.TotalFee // 全额退款
	} else {
		req.BizContent.TotalFee = -req.BizContent.RefundFee // 退款改为负数金额
	}
	refundOrder, err := srv.handerOrder(&orderPB.Order{
		StoreId:    req.StoreId,                // 商户门店编号 收款账号ID userID
		Channel:    originalOrder.Channel,      // 付款方式 [支付宝、微信、银联等]
		AuthCode:   originalOrder.AuthCode,     // 付款码
		Title:      originalOrder.Title,        // 订单标题
		TotalFee:   req.BizContent.TotalFee,    // 订单总金额
		OutTradeNo: req.BizContent.OutRefundNo, // 订单编号
		OperatorId: originalOrder.OperatorId,   // 商户操作员编号
		TerminalId: originalOrder.TerminalId,   // 商户机具终端编号
		LinkId:     originalOrder.Id,
		Status:     0,                     // 订单状态 默认状态未付款
		Attach:     req.BizContent.Attach, // 商户机具终端编号
	}) //创建退款订单返回订单ID
	res.Content = srv.handerRefund(refundOrder, originalOrder)
	return nil
}

// // AffirmRefund 确认退款
// func (srv *Trade) AffirmRefund(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
// 	res.Error = &pd.Error{}
// 	config, err := srv.userConfig(req.BizContent) // 获取配置同时根据商家名称赋值商家id
// 	if err != nil {
// 		res.Error.Code = "AffirmRefund.userConfig"
// 		res.Error.Detail = "退款是支付配置信息查询失败"
// 		log.Fatal(req, res, err)
// 		return nil
// 	}
// 	refundOrder, err := srv.getOrder(req.BizContent) //创建订单返回订单ID
// 	if err != nil {
// 		res.Error.Code = "AffirmRefund.getOrder"
// 		res.Error.Detail = "确认退款获取订单失败"
// 		log.Fatal(req, res, err)
// 		return nil
// 	}
// 	originalOrder := &orderPB.Order{
// 		Id: refundOrder.LinkId,
// 	}
// 	err = srv.Repo.Get(originalOrder)
// 	if err != nil {
// 		res.Error.Code = "AffirmRefund.Repo.Get"
// 		res.Error.Detail = "确认退款获取原始订单失败"
// 		log.Fatal(req, res, err)
// 		return nil
// 	}
// 	res.Valid, res.Content, err = srv.handerRefund(config, refundOrder, originalOrder)
// 	return
// }

// handerRefund 处理退款
func (srv *Trade) handerRefund(refundOrder *orderPB.Order, originalOrder *orderPB.Order) (resContent *pd.Content) {
	content := mxj.New()
	resContent = &pd.Content{}
	var err error
	switch refundOrder.Channel {
	case "alipay":
		srv.newAlipayClient() //实例化支付宝连接
		content, err = srv.Alipay.Refund(refundOrder, originalOrder)
	case "wechat":
		srv.newWechatClient() //实例化微信连接
		content, err = srv.Wechat.Refund(refundOrder, originalOrder)
	case "icbc":
		srv.newIcbcClient() //实例化微信连接
		content, err = srv.Icbc.Refund(refundOrder, originalOrder)
	}
	if err != nil {
		resContent.ReturnCode = "Refund." + refundOrder.Channel + ".Error"
		resContent.ReturnMsg = refundOrder.Channel + "退款请求失败:" + err.Error()
		log.Fatal(resContent, err)
		return resContent
	}
	resContent.ReturnCode = content["return_code"].(string)
	resContent.ReturnMsg = content["return_msg"].(string)
	if content["return_code"] == "SUCCESS" {
		err = srv.successOrder(refundOrder)
		if err != nil {
			resContent.Status = WAITING
			resContent.ReturnCode = "Refund.Success.Update"
			resContent.ReturnMsg = "退款成功,更新订单状态失败!"
			return resContent
		}
		err = srv.Repo.UpdateRefundFee(originalOrder)
		if err != nil {
			resContent.Status = WAITING
			resContent.ReturnCode = "Refund.Success.UpdateRefundFee"
			resContent.ReturnMsg = "退款成功,更新原始订单退款金额失败!"
			return resContent
		}
		resContent.Status = SUCCESS
		resContent.Channel = content["channel"].(string)
		resContent.OutTradeNo = originalOrder.OutTradeNo
		resContent.OutRefundNo = refundOrder.OutTradeNo
		resContent.TradeNo = content["trade_no"].(string)
		resContent.TotalFee = originalOrder.TotalFee
		resContent.RefundFee = -refundOrder.TotalFee

	}
	resContent.ReturnMsg = content["return_msg"].(string)
	c, _ := content["content"].(mxj.Map).Json()
	resContent.Content = string(c)
	return resContent
}

// RefundQuery 退款查询
func (srv *Trade) RefundQuery(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	// 初始化回参
	res.Content = &pd.Content{}
	if req.BizContent.OutRefundNo == "" {
		res.Content.ReturnCode = "RefundQuery.OutRefundNo.Not"
		res.Content.ReturnMsg = "退款订单号不允许为空:BizContent.OutRefundNo"
		return
	}
	if req.BizContent.OutTradeNo == "" {
		res.Content.ReturnCode = "RefundQuery.OutTradeNo.Not"
		res.Content.ReturnMsg = "订单编号不允许为空:BizContent.OutTradeNo"
		return
	}

	err = srv.userConfig(req)
	if err != nil {
		res.Content.ReturnCode = "RefundQuery.userConfig"
		res.Content.ReturnMsg = "查询商户支付配置信息失败"
		log.Fatal(req, res, err)
		return nil
	}
	if !srv.con.Status {
		res.Content.ReturnCode = "RefundQuery.Status"
		res.Content.ReturnMsg = "支付功能被禁用！请联系管理员。"
		log.Fatal(req, res, err)
		return nil
	}
	originalOrder := &orderPB.Order{
		StoreId:    req.StoreId,               // 商户门店编号 收款账号ID userID
		OutTradeNo: req.BizContent.OutTradeNo, // 订单编号
	}

	err = srv.Repo.StoreIdAndOutTradeNoGet(originalOrder) //获取订单返回订单ID
	if err != nil {
		res.Content.Status = CLOSED
		res.Content.ReturnCode = "RefundQuery.OriginalOrder.StoreIdAndOutTradeNoGet"
		res.Content.ReturnMsg = "获取订单失败,获取退款订单信息失败"
		log.Fatal(req, res, err)
		return nil
	}
	repoOrder := &orderPB.Order{
		StoreId:    req.StoreId,                // 商户门店编号 收款账号ID userID
		OutTradeNo: req.BizContent.OutRefundNo, // 退款订单编号
	}

	err = srv.Repo.StoreIdAndOutTradeNoGet(repoOrder) //获取订单返回订单ID
	if err != nil {
		res.Content.ReturnCode = "RefundQuery.repoOrder.StoreIdAndOutTradeNoGet"
		res.Content.ReturnMsg = "获取订单失败,获取订单相关退款订单信息失败"
		log.Fatal(req, res, err)
		return nil
	}

	if repoOrder.TotalFee > 0 { // 退款查询时不进行是不进行实际查询等待系统自动结果
		res.Content.ReturnCode = "RefundQuery.Query.Order"
		res.Content.ReturnMsg = "普通订单请使用查询接口查询订单"
		log.Fatal(req, res, err)
		return nil
	}
	content := mxj.New()
	switch repoOrder.Channel {
	case "alipay":
		srv.newAlipayClient() //实例化支付宝连接
		content, err = srv.Alipay.RefundQuery(req.BizContent)
	case "wechat":
		srv.newWechatClient() //实例化微信连接
		content, err = srv.Wechat.RefundQuery(req.BizContent)
	case "icbc":
		srv.newIcbcClient() //实例化微信连接
		content, err = srv.Icbc.RefundQuery(req.BizContent)
	}
	if err != nil {
		res.Content.ReturnCode = "RefundQuery." + req.BizContent.Channel + ".Error"
		res.Content.ReturnMsg = req.BizContent.Channel + "退款查询请求失败:" + err.Error()
		log.Fatal(res, err)
		return nil
	}
	srv.handerRefundQuery(content, res, repoOrder, originalOrder)
	return nil
}

// userConfig 用户配置
func (srv *Trade) userConfig(req *pd.Request) error {
	if req.StoreId == "" {
		return errors.New("商户ID不允许为空:StoreId")
	}
	config := &configPB.Config{
		Id: req.StoreId,
	}
	err := srv.Config.Get(config)
	if err != nil {
		return err
	}
	// 通道设置
	if req.BizContent.Channel == "" {
		// 默认通道
		if config.Channel != "" && config.Channel != "wechat" && config.Channel != "alipay" {
			req.BizContent.Channel = config.Channel
		} else {
			// 直连通道
			if ok, _ := regexp.Match(`^(?:2[5-9]|30)\d{14,18}$`, []byte(req.BizContent.AuthCode)); ok {
				req.BizContent.Channel = "alipay"
			}
			if ok, _ := regexp.Match(`^1[0-5]\d{16}$`, []byte(req.BizContent.AuthCode)); ok {
				req.BizContent.Channel = "wechat"
			}
		}
	}
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
	if config.Icbc.SubMerId != "" { // 子商户模式需要通过系统配置进行设置服务商信息
		config.Icbc.AppId = env.Getenv("PAY_ICBC_APPID", config.Icbc.AppId)
		config.Icbc.PrivateKey = env.Getenv("PAY_ICBC_PRIVATE_KEY", config.Icbc.PrivateKey)
		config.Icbc.IcbcPublicKey = env.Getenv("PAY_ICBC_ICBC_PUBLIC_KEY", config.Icbc.IcbcPublicKey)
		config.Icbc.SignType = env.Getenv("PAY_ICBC_SIGN_TYPE", config.Icbc.SignType)
		config.Icbc.ReturnSignType = env.Getenv("PAY_ICBC_RETURN_SIGN_TYPE", config.Icbc.ReturnSignType)
		config.Icbc.MerId = config.Icbc.SubMerId
	}
	srv.con = config
	return err
}

// getOrder 获取订单
func (srv *Trade) getOrder(b *pd.Request) (repoOrder *orderPB.Order, err error) {
	repoOrder = &orderPB.Order{
		StoreId:    b.StoreId,               // 商户门店编号 收款账号ID userID
		OutTradeNo: b.BizContent.OutTradeNo, // 订单编号
	}
	err = srv.Repo.StoreIdAndOutTradeNoGet(repoOrder)
	return repoOrder, err
}

// handerOrder 处理订单
func (srv *Trade) handerOrder(repoOrder *orderPB.Order) (*orderPB.Order, error) {
	err := srv.Repo.StoreIdAndOutTradeNoGet(repoOrder)
	if err == gorm.ErrRecordNotFound {
		err = srv.Repo.Create(repoOrder)
		if err != nil {
			return nil, err
		}
	}
	return repoOrder, err
}

// newAlipayClient 实例化支付宝付款方式连接
func (srv *Trade) newAlipayClient() {
	srv.Alipay.NewClient(map[string]string{
		"AppId":                srv.con.Alipay.AppId,
		"PrivateKey":           srv.con.Alipay.PrivateKey,
		"AliPayPublicKey":      srv.con.Alipay.AliPayPublicKey,
		"AppAuthToken":         srv.con.Alipay.AppAuthToken,
		"SysServiceProviderId": srv.con.Alipay.SysServiceProviderId,
		"SignType":             srv.con.Alipay.SignType,
	}, srv.con.Alipay.Sandbox)
}

// newWechatClient 实例化微信付款方式连接
func (srv *Trade) newWechatClient() {
	srv.Wechat.NewClient(map[string]string{
		"AppId":    srv.con.Wechat.AppId,
		"MchId":    srv.con.Wechat.MchId,
		"ApiKey":   srv.con.Wechat.ApiKey,
		"SubAppId": srv.con.Wechat.SubAppId,
		"SubMchId": srv.con.Wechat.SubMchId,
		"PemCert":  srv.con.Wechat.PemCert,
		"PemKey":   srv.con.Wechat.PemKey,
	}, srv.con.Wechat.Sandbox)
}

// newIcbcClient 实例化微信付款方式连接
func (srv *Trade) newIcbcClient() {
	srv.Icbc.NewClient(map[string]string{
		"AppId":          srv.con.Icbc.AppId,
		"PrivateKey":     srv.con.Icbc.PrivateKey,
		"IcbcPublicKey":  srv.con.Icbc.IcbcPublicKey,
		"SignType":       srv.con.Icbc.SignType,
		"ReturnSignType": srv.con.Icbc.ReturnSignType,
		"MerId":          srv.con.Icbc.MerId,
	})
}

// handerAopF2F 处理扫码支付回调信息
func (srv *Trade) handerAopF2F(content mxj.Map, res *pd.Response, repoOrder *orderPB.Order) (err error) {
	res.Content.Channel = repoOrder.Channel
	res.Content.ReturnCode = content["return_code"].(string)
	res.Content.ReturnMsg = content["return_msg"].(string)
	if content["return_code"] == "SUCCESS" {
		repoOrder.TradeNo = content["trade_no"].(string)
		err = srv.successOrder(repoOrder)
		if err != nil {
			res.Content.Status = WAITING
			res.Content.ReturnCode = "AopF2.Update.Success"
			res.Content.ReturnMsg = "支付成功,更新订单状态失败!"
		}
		res.Content.Status = SUCCESS
		res.Content.Channel = content["channel"].(string)
		res.Content.OutTradeNo = content["out_trade_no"].(string)
		res.Content.TradeNo = content["trade_no"].(string)
		res.Content.TotalFee = repoOrder.TotalFee
		res.Content.RefundFee = repoOrder.RefundFee
		res.Content.TimeEnd = content["time_end"].(string)
		if v, ok := content["wechat_open_id"]; ok {
			res.Content.WechatOpenId = v.(string)
		}
		if v, ok := content["wechat_is_subscribe"]; ok {
			res.Content.WechatIsSubscribe = v.(string)
		}
		if v, ok := content["alipay_buyer_logon_id"]; ok {
			res.Content.AlipayBuyerLogonId = v.(string)
		}
		if v, ok := content["alipay_buyer_user_id"]; ok {
			res.Content.AlipayBuyerUserId = v.(string)
		}
	}
	c, _ := content["content"].(mxj.Map).Json()
	res.Content.Content = string(c)
	return nil
}

// handerQuery 处理订单查询支付回调信息
func (srv *Trade) handerQuery(content mxj.Map, res *pd.Response, repoOrder *orderPB.Order) (err error) {
	res.Content.ReturnCode = content["return_code"].(string)
	res.Content.ReturnMsg = content["return_msg"].(string)
	res.Content.Channel = repoOrder.Channel
	if content["return_code"] == "SUCCESS" {
		switch content["status"].(string) {
		case SUCCESS:
			repoOrder.TradeNo = content["trade_no"].(string)
			err = srv.successOrder(repoOrder)
			if err != nil {
				res.Content.Status = WAITING
				res.Content.ReturnCode = "Query.Update.Success"
				res.Content.ReturnMsg = "支付成功,更新订单状态失败!"
			}
			res.Content.Status = SUCCESS
		case CLOSED:
			repoOrder.Fee = 0
			repoOrder.Status = -1
			err = srv.Repo.Update(repoOrder)
			if err != nil {
				res.Content.Status = WAITING
				res.Content.ReturnCode = "Query.Update.Close"
				res.Content.ReturnMsg = "支付失败,更新订单状态失败!"
			}
			res.Content.Status = CLOSED
		case USERPAYING:
			res.Content.Status = USERPAYING
		case WAITING:
			res.Content.Status = WAITING
		}
		res.Content.Channel = content["channel"].(string)
		res.Content.OutTradeNo = content["out_trade_no"].(string)
		res.Content.TradeNo = content["trade_no"].(string)
		res.Content.TotalFee = repoOrder.TotalFee
		res.Content.RefundFee = repoOrder.RefundFee
		if v, ok := content["wechat_open_id"]; ok {
			res.Content.WechatOpenId = v.(string)
		}
		if v, ok := content["wechat_is_subscribe"]; ok {
			res.Content.WechatIsSubscribe = v.(string)
		}
		if v, ok := content["alipay_buyer_logon_id"]; ok {
			res.Content.AlipayBuyerLogonId = v.(string)
		}
		if v, ok := content["alipay_buyer_user_id"]; ok {
			res.Content.AlipayBuyerUserId = v.(string)
		}
	}
	c, _ := content["content"].(mxj.Map).Json()
	res.Content.Content = string(c)
	return nil
}

// handerRefundQuery 处理订单退款查询支付回调信息
func (srv *Trade) handerRefundQuery(content mxj.Map, res *pd.Response, refundOrder *orderPB.Order, originalOrder *orderPB.Order) (err error) {
	res.Content.Status = WAITING
	res.Content.ReturnCode = content["return_code"].(string)
	res.Content.ReturnMsg = content["return_msg"].(string)
	if content["return_code"] == "SUCCESS" {
		err = srv.successOrder(refundOrder)
		if err != nil {
			res.Content.Status = WAITING
			res.Content.ReturnCode = "Refund.Success.Update"
			res.Content.ReturnMsg = "退款成功,更新订单状态失败!"
			return
		}
		err = srv.Repo.UpdateRefundFee(originalOrder)
		if err != nil {
			res.Content.Status = WAITING
			res.Content.ReturnCode = "Refund.Success.UpdateRefundFee"
			res.Content.ReturnMsg = "退款成功,更新原始订单退款金额失败!"
			return
		}
		res.Content.Status = SUCCESS
		res.Content.Channel = content["channel"].(string)
		res.Content.OutTradeNo = originalOrder.OutTradeNo
		res.Content.OutRefundNo = refundOrder.OutTradeNo
		res.Content.TradeNo = content["trade_no"].(string)
		res.Content.TotalFee = originalOrder.TotalFee
		res.Content.RefundFee = -refundOrder.TotalFee
	}
	c, _ := content["content"].(mxj.Map).Json()
	res.Content.Content = string(c)
	return nil
}

// successOrder 支付成功订单处理
func (srv *Trade) successOrder(repoOrder *orderPB.Order) (err error) {
	var fee int64
	switch repoOrder.Channel {
	case "alipay":
		fee = srv.con.Alipay.Fee
	case "wechat":
		fee = srv.con.Wechat.Fee
	case "icbc":
		fee = srv.con.Icbc.Fee
	}
	repoOrder.Status = 1
	repoOrder.Fee = int64(math.Floor(float64(repoOrder.TotalFee*fee)/10000 + 0.5)) // 相乘后转浮点型乘以万分之一然后四舍五入 【+0.5四舍五入取整】
	err = srv.Repo.Update(repoOrder)
	if err != nil {
		return err
	}
	return err
}
