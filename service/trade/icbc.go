package trade

import (
	"strconv"
	"time"

	"github.com/clbanning/mxj"
	notifyPB "github.com/lecex/pay/proto/notify"
	orderPB "github.com/lecex/pay/proto/order"
	proto "github.com/lecex/pay/proto/trade"

	// "github.com/shopspring/decimal"

	"github.com/bigrocs/icbc"
	"github.com/bigrocs/icbc/requests"
)

type Icbc struct {
	Client *icbc.Client
	config map[string]string
}

func (srv *Icbc) NewClient(config map[string]string) {
	srv.config = config
	srv.Client = icbc.NewClient()
	c := srv.Client.Config
	c.AppId = config["AppId"]
	c.PrivateKey = config["PrivateKey"]
	c.IcbcPublicKey = config["IcbcPublicKey"]
	c.SignType = config["SignType"]
	c.ReturnSignType = config["ReturnSignType"]
}

// Query 支付查询
func (srv *Icbc) Query(b *proto.BizContent) (req mxj.Map, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.ApiName = "pay.orderquery"
	request.BizContent = map[string]interface{}{
		"mer_id":       srv.config["MerId"],
		"out_trade_no": b.OutTradeNo,
	}
	return srv.request(request)
}

// AopF2F 商家扫用户付款码
//    文档地址：https://docs.open.icbc.com/api_1/icbc.trade.pay
func (srv *Icbc) AopF2F(b *proto.BizContent) (req mxj.Map, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.ApiName = "pay.pay"
	request.BizContent = map[string]interface{}{
		"mer_id":       srv.config["MerId"],
		"qr_code":      b.AuthCode,
		"out_trade_no": b.OutTradeNo,
		"order_amt":    strconv.FormatInt(b.TotalFee, 10),
		"trade_date":   time.Now().Format("20060102"),
		"trade_time":   time.Now().Format("150405"),
	}
	return srv.request(request)
}

// Refund 交易退款
//    文档地址：https://opendocs.icbc.com/apis/api_1/icbc.trade.refund/
func (srv *Icbc) Refund(refundOrder *orderPB.Order, originalOrder *orderPB.Order) (req mxj.Map, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.ApiName = "pay.refund"
	request.BizContent = map[string]interface{}{
		"mer_id":       srv.config["MerId"],
		"out_trade_no": originalOrder.OutTradeNo,
		"reject_no":    refundOrder.OutTradeNo,
		"reject_amt":   strconv.FormatInt(-refundOrder.TotalFee, 10),
	}
	return srv.request(request)
}

// RefundQuery 退款查询
func (srv *Icbc) RefundQuery(b *proto.BizContent) (req mxj.Map, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.ApiName = "pay.refundquery"
	request.BizContent = map[string]interface{}{
		"mer_id":       srv.config["MerId"],
		"out_trade_no": b.OutTradeNo,
		"reject_no":    b.OutRefundNo, //debug
	}
	return srv.request(request)
}

// request 请求处理
func (srv *Icbc) request(request *requests.CommonRequest) (req mxj.Map, err error) {
	response, err := srv.Client.ProcessCommonRequest(request)
	if err != nil {
		return req, err
	}
	return response.GetVerifySignDataMap()
}

// Notify 异步通知
func (srv *Icbc) Notify(req *notifyPB.Request) (ok bool, err error) {
	// // ====异步通知参数解析和验签Sign====
	// // 解析异步通知的参数
	// //    req：*http.Request
	// //    返回参数 notifyReq：通知的参数
	// //    返回参数 err：错误信息
	// notifyReq, err := srv.parseNotifyResult(req) // c.Request()是 echo 框架的获取
	// // 验签操作
	// return icbc.VerifySign(srv.config["icbcPublicKey"], notifyReq)
	return
}
