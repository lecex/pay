package service

import (
	"strconv"

	orderPB "github.com/lecex/pay/proto/order"
	proto "github.com/lecex/pay/proto/pay"

	"github.com/bigrocs/wechat"
	"github.com/bigrocs/wechat/requests"
	"github.com/bigrocs/wechat/util"

	"github.com/clbanning/mxj"
)

type Wechat struct {
	Client *wechat.Client
	config map[string]string
}

// NewClient 创建新的连接
func (srv *Wechat) NewClient(config map[string]string, sandbox bool) {
	srv.config = config
	srv.Client = wechat.NewClient()
	c := srv.Client.Config
	c.AppId = config["AppId"]
	c.MchId = config["MchId"]
	c.ApiKey = config["ApiKey"]
	c.SubAppId = config["SubAppId"]
	c.SubMchId = config["SubMchId"]
	c.PemCert = config["PemCert"]
	c.PemKey = config["PemKey"]
}

// Query 支付查询
func (srv *Wechat) Query(order *proto.Order) (req mxj.Map, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.Domain = "mch"
	request.ApiName = "pay.orderquery"
	request.QueryParams = map[string]interface{}{
		"out_trade_no": order.OrderNo,
	}
	// 请求
	return srv.request(request)
}

// AopF2F 商家扫用户付款码
//    文档地址：https://pay.weixin.qq.com/wiki/doc/api/micropay_sl.php?chapter=9_10&index=1
func (srv *Wechat) AopF2F(order *proto.Order) (req mxj.Map, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.Domain = "mch"
	request.ApiName = "pay.micropay"
	request.QueryParams = map[string]interface{}{
		"auth_code":        order.AuthCode,
		"body":             order.Title,
		"out_trade_no":     order.OrderNo,
		"total_fee":        strconv.FormatInt(order.TotalAmount, 10),
		"spbill_create_ip": "127.0.0.1",
	}
	// 请求
	return srv.request(request)

}

// Cancel 撤销交易
//    文档地址：https://pay.weixin.qq.com/wiki/doc/api/micropay.php?chapter=9_11&index=3
func (srv *Wechat) Cancel(order *proto.Order) (req mxj.Map, err error) {
	request := requests.NewCommonRequest()
	request.Domain = "mch"
	request.ApiName = "pay.reverse"
	request.QueryParams = map[string]interface{}{
		"out_trade_no": order.OrderNo,
	}
	return srv.request(request)
}

// Refund 交易退款
//    文档地址：https://pay.weixin.qq.com/wiki/doc/api/micropay.php?chapter=9_4
func (srv *Wechat) Refund(refundOrder *orderPB.Order, originalOrder *orderPB.Order) (req mxj.Map, err error) {
	request := requests.NewCommonRequest()
	request.Domain = "mch"
	request.ApiName = "pay.refund"
	request.QueryParams = map[string]interface{}{
		"out_trade_no":  originalOrder.OrderNo,
		"out_refund_no": refundOrder.OrderNo,
		"total_fee":     strconv.FormatInt(originalOrder.TotalAmount, 10),
		"refund_fee":    strconv.FormatInt(-refundOrder.TotalAmount, 10),
	}
	return srv.request(request)
}

func (srv *Wechat) request(request *requests.CommonRequest) (req mxj.Map, err error) {
	// 请求
	response, err := srv.Client.ProcessCommonRequest(request)
	if err != nil {
		return req, err
	}
	req, err = response.GetHttpContentMap()
	if err != nil {
		return req, err
	}
	if req["return_code"] == "SUCCESS" {
		ok := util.VerifySign(req, srv.config["ApiKey"], util.SignType_MD5)
		if ok && err == nil {
			return req, err
		}
	}
	return req, err
}

// Notify 异步通知
func (srv *Wechat) Notify(body string) (ok bool, err error) {
	// notifyReq, err := srv.ParseNotifyResult(body)
	// if err != nil {
	// 	return ok, err
	// }
	// return util.VerifySign(notifyReq, srv.config["ApiKey"], util.SignType_MD5), err
	return
}

// ParseNotifyResult 解析异步通知
func (srv *Wechat) ParseNotifyResult(body string) (rsp map[string]interface{}, err error) {
	mv, err := mxj.NewMapXml([]byte(body))
	return mv["xml"].(map[string]interface{}), err
}
