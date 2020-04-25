package service

import (
	"errors"

	"github.com/clbanning/mxj"
	notifyPB "github.com/lecex/pay/proto/notify"
	proto "github.com/lecex/pay/proto/pay"
	"github.com/shopspring/decimal"

	// "github.com/shopspring/decimal"

	"github.com/bigrocs/alipay"
	"github.com/bigrocs/alipay/requests"
	"github.com/bigrocs/alipay/util"
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
	c.Sandbox = sandbox
}

// Query 支付查询
func (srv *Alipay) Query(order *proto.Order) (req mxj.Map, err error) {
	c := srv.Client.Config
	c.Method = "alipay.trade.query"
	// 配置参数
	request := requests.NewCommonRequest()
	request.BizContent = map[string]interface{}{
		"out_trade_no": order.OrderNo,
	}
	return srv.request(request)
}

// AopF2F 商家扫用户付款码
//    文档地址：https://docs.open.alipay.com/api_1/alipay.trade.pay
func (srv *Alipay) AopF2F(order *proto.Order) (req mxj.Map, err error) {
	c := srv.Client.Config
	c.Method = "alipay.trade.pay"
	// 配置参数
	request := requests.NewCommonRequest()
	request.BizContent = map[string]interface{}{
		"subject":         order.Title,
		"scene":           "bar_code",
		"auth_code":       order.AuthCode,
		"out_trade_no":    order.OrderNo,
		"total_amount":    decimal.NewFromFloat(float64(order.TotalAmount)).Div(decimal.NewFromFloat(float64(100))),
		"timeout_express": "30m",
		"extend_params":   map[string]interface{}{"sys_service_provider_id": srv.config["SysServiceProviderId"]},
	}
	return srv.request(request)
}

// request 请求处理
func (srv *Alipay) request(request *requests.CommonRequest) (req mxj.Map, err error) {
	response, err := srv.Client.ProcessCommonRequest(request)
	if err != nil {
		return req, err
	}
	req, err = response.GetHttpContentMap()
	if ok, err := util.VerifySign(response.GetSignData(), req["sign"].(string), srv.config["AliPayPublicKey"], srv.config["SignType"]); !ok {
		if err != nil {
			return req, err
		}
		return req, errors.New("返回数据 Sign 校验失败")
	}
	return response.GetSignDataMap()
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
