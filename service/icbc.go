package service

import (
	"strconv"
	"time"

	"github.com/clbanning/mxj"
	notifyPB "github.com/lecex/pay/proto/notify"
	orderPB "github.com/lecex/pay/proto/order"
	proto "github.com/lecex/pay/proto/pay"

	// "github.com/shopspring/decimal"

	"github.com/bigrocs/icbc"
	"github.com/bigrocs/icbc/requests"
	"github.com/bigrocs/icbc/util"
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
	c.SignType = config["SignType"]
}

// Query 支付查询
func (srv *Icbc) Query(order *proto.Order) (req mxj.Map, err error) {
	c := srv.Client.Config
	c.ApiName = "pay.orderquery"
	// 配置参数
	request := requests.NewCommonRequest()
	request.BizContent = map[string]interface{}{
		"mer_id":       srv.config["MerId"],
		"out_trade_no": order.OrderNo,
	}
	return srv.request(request)
}

// AopF2F 商家扫用户付款码
//    文档地址：https://docs.open.icbc.com/api_1/icbc.trade.pay
func (srv *Icbc) AopF2F(order *proto.Order) (req mxj.Map, err error) {
	c := srv.Client.Config
	c.ApiName = "pay.pay"
	// 配置参数
	request := requests.NewCommonRequest()
	request.BizContent = map[string]interface{}{
		"mer_id":       srv.config["MerId"],
		"qr_code":      order.AuthCode,
		"out_trade_no": order.OrderNo,
		"order_amt":    strconv.FormatInt(order.TotalAmount, 10),
		"trade_date":   time.Now().Format("20060102"),
		"trade_time":   time.Now().Format("150405"),
	}
	return srv.request(request)
}

// Refund 交易退款
//    文档地址：https://opendocs.icbc.com/apis/api_1/icbc.trade.refund/
func (srv *Icbc) Refund(refundOrder *orderPB.Order, originalOrder *orderPB.Order) (req mxj.Map, err error) {
	c := srv.Client.Config
	c.ApiName = "pay.refund"
	// 配置参数
	request := requests.NewCommonRequest()
	request.BizContent = map[string]interface{}{
		"mer_id":        srv.config["MerId"],
		"out_trade_no":  originalOrder.OrderNo,
		"reject_no":     refundOrder.OrderNo,
		"refund_amount": strconv.FormatInt(refundOrder.TotalAmount, 10),
	}
	return srv.request(request)
}

// request 请求处理
func (srv *Icbc) request(request *requests.CommonRequest) (req mxj.Map, err error) {
	response, err := srv.Client.ProcessCommonRequest(request)
	if err != nil {
		return req, err
	}
	req, err = response.GetHttpContentMap()
	if err != nil {
		return req, err
	}
	if req["sign"] == nil {
		return response.GetSignDataMap()
	}
	ok, err := util.VerifySign(response.GetSignData(), req["sign"].(string), srv.config["IcbcPublicKey"], srv.config["ReturnSignType"])
	if ok && err == nil {
		return response.GetSignDataMap()
	}
	return req, err
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
