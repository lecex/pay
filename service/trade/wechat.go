package trade

import (
	"strconv"
	"time"

	orderPB "github.com/lecex/pay/proto/order"
	proto "github.com/lecex/pay/proto/trade"

	"github.com/bigrocs/wechat"
	"github.com/bigrocs/wechat/requests"

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
func (srv *Wechat) Query(b *proto.BizContent) (req mxj.Map, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.Domain = "mch"
	request.ApiName = "pay.orderquery"
	request.QueryParams = map[string]interface{}{
		"out_trade_no": b.OutTradeNo,
	}
	// 请求
	return srv.request(request)
}

// AopF2F 商家扫用户付款码
//    文档地址：https://pay.weixin.qq.com/wiki/doc/api/micropay_sl.php?chapter=9_10&index=1
func (srv *Wechat) AopF2F(b *proto.BizContent) (req mxj.Map, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.Domain = "mch"
	request.ApiName = "pay.micropay"
	request.QueryParams = map[string]interface{}{
		"auth_code":        b.AuthCode,
		"body":             b.Title,
		"out_trade_no":     b.OutTradeNo,
		"total_fee":        strconv.FormatInt(b.TotalFee, 10),
		"spbill_create_ip": "127.0.0.1",
		"time_start":       time.Now().Format("20060102150405"),                      // 当前时间
		"time_expire":      time.Now().Add(time.Minute * 2).Format("20060102150405"), // 二分钟后结束
	}
	// 请求
	return srv.request(request)

}

// Cancel 撤销交易
//    文档地址：https://pay.weixin.qq.com/wiki/doc/api/micropay.php?chapter=9_11&index=3
func (srv *Wechat) Cancel(b *proto.BizContent) (req mxj.Map, err error) {
	request := requests.NewCommonRequest()
	request.Domain = "mch"
	request.ApiName = "pay.reverse"
	request.QueryParams = map[string]interface{}{
		"out_trade_no": b.OutTradeNo,
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
		"out_trade_no":  originalOrder.OutTradeNo,
		"out_refund_no": refundOrder.OutTradeNo,
		"total_fee":     strconv.FormatInt(originalOrder.TotalFee, 10),
		"refund_fee":    strconv.FormatInt(-refundOrder.TotalFee, 10),
	}
	return srv.request(request)
}

func (srv *Wechat) request(request *requests.CommonRequest) (req mxj.Map, err error) {
	// 请求
	response, err := srv.Client.ProcessCommonRequest(request)
	if err != nil {
		return req, err
	}
	return response.GetVerifySignDataMap()
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
