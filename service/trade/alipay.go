package trade

import (
	"github.com/clbanning/mxj"
	notifyPB "github.com/lecex/pay/proto/notify"
	orderPB "github.com/lecex/pay/proto/order"
	proto "github.com/lecex/pay/proto/trade"
	"github.com/shopspring/decimal"

	// "github.com/shopspring/decimal"

	"github.com/bigrocs/alipay"
	"github.com/bigrocs/alipay/requests"
)

type Alipay struct {
	Client *alipay.Client
	config map[string]string
}

func (srv *Alipay) NewClient(config map[string]string, sandbox bool) {
	srv.config = config
	srv.Client = alipay.NewClient()
	c := srv.Client.Config
	c.AppId = config["AppId"]
	c.PrivateKey = config["PrivateKey"]
	c.SignType = config["SignType"]
	c.AppAuthToken = config["AppAuthToken"]
	c.AliPayPublicKey = config["AliPayPublicKey"]
	c.Sandbox = sandbox
}

// Query 支付查询
func (srv *Alipay) Query(b *proto.BizContent) (req mxj.Map, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.ApiName = "alipay.trade.query"
	request.BizContent = map[string]interface{}{
		"out_trade_no": b.OutTradeNo,
	}
	return srv.request(request)
}

// AopF2F 商家扫用户付款码
//    文档地址：https://docs.open.alipay.com/api_1/alipay.trade.pay
func (srv *Alipay) AopF2F(b *proto.BizContent) (req mxj.Map, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.ApiName = "alipay.trade.pay"
	request.BizContent = map[string]interface{}{
		"subject":         b.Title,
		"scene":           "bar_code",
		"auth_code":       b.AuthCode,
		"out_trade_no":    b.OutTradeNo,
		"total_amount":    decimal.NewFromFloat(float64(b.TotalFee)).Div(decimal.NewFromFloat(float64(100))),
		"timeout_express": "2m",
		"extend_params":   map[string]interface{}{"sys_service_provider_id": srv.config["SysServiceProviderId"]},
	}
	return srv.request(request)
}

// Cancel 撤销交易
//    文档地址：https://opendocs.alipay.com/apis/api_1/alipay.trade.cancel/
func (srv *Alipay) Cancel(b *proto.BizContent) (req mxj.Map, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.ApiName = "alipay.trade.cancel"
	request.BizContent = map[string]interface{}{
		"out_trade_no": b.OutTradeNo,
	}
	return srv.request(request)
}

// Refund 交易退款
//    文档地址：https://opendocs.alipay.com/apis/api_1/alipay.trade.refund/
func (srv *Alipay) Refund(refundOrder *orderPB.Order, originalOrder *orderPB.Order) (req mxj.Map, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.ApiName = "alipay.trade.refund"
	request.BizContent = map[string]interface{}{
		"out_trade_no":   originalOrder.OutTradeNo,
		"out_request_no": refundOrder.OutTradeNo,
		"refund_amount":  decimal.NewFromFloat(float64(-refundOrder.TotalFee)).Div(decimal.NewFromFloat(float64(100))),
	}
	return srv.request(request)
}

// request 请求处理
func (srv *Alipay) request(request *requests.CommonRequest) (req mxj.Map, err error) {
	response, err := srv.Client.ProcessCommonRequest(request)
	if err != nil {
		return nil, err
	}
	return response.GetVerifySignDataMap()
}

// Notify 异步通知
//    文档地址：https://opendocs.alipay.com/open/200/106120
func (srv *Alipay) Notify(req *notifyPB.Request) (ok bool, err error) {
	// // ====异步通知参数解析和验签Sign====
	// // 解析异步通知的参数
	// //    req：*http.Request
	// //    返回参数 notifyReq：通知的参数
	// //    返回参数 err：错误信息
	// notifyReq, err := srv.parseNotifyResult(req) // c.Request()是 echo 框架的获取
	// // 验签操作
	// return alipay.VerifySign(srv.config["AliPayPublicKey"], notifyReq)
	return
}
