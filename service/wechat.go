package service

import (
	"encoding/xml"
	"fmt"
	"strconv"

	notifyPB "github.com/lecex/pay/proto/notify"
	proto "github.com/lecex/pay/proto/pay"

	"github.com/iGoogle-ink/gopay"
	"github.com/iGoogle-ink/gopay/wechat"
)

type Wechat struct {
	Client *wechat.Client
	config map[string]string
}

func (srv *Wechat) NewClient(config map[string]string, sandbox bool) {
	srv.config = config
	srv.Client = &wechat.Client{
		AppId:  config["AppId"],
		MchId:  config["MchId"],
		ApiKey: config["ApiKey"],
		// SubAppId: config["SubAppId"],
		// SubMchId: config["SubMchId"],
		IsProd: !sandbox,
	}
}

// AopF2F 商家扫用户付款码
//    文档地址：https://pay.weixin.qq.com/wiki/doc/api/micropay_sl.php?chapter=9_10&index=1
func (srv *Wechat) AopF2F(order *proto.Order) (ok bool, err error) {
	body := make(gopay.BodyMap)
	body.Set("sub_appid", srv.config["SubAppId"])
	body.Set("sub_mch_id", srv.config["SubMchId"])
	body.Set("nonce_str", gopay.GetRandomString(32))
	body.Set("auth_code", order.AuthCode)
	body.Set("body", order.Title)
	body.Set("out_trade_no", order.OrderNo)
	body.Set("total_fee", strconv.FormatInt(order.TotalAmount, 10))
	body.Set("timeout_express", "2m")
	body.Set("spbill_create_ip", "127.0.0.1")
	wxRsp, err := srv.Client.UnifiedOrder(body)
	if err != nil {
		return ok, err
	}
	return wechat.VerifySign(srv.config["ApiKey"], wechat.SignType_MD5, wxRsp)

}

// Notify 异步通知
func (srv *Wechat) Notify(req *notifyPB.Request) (xml string, err error) {
	// ====支付异步通知参数解析和验签Sign====
	// 解析支付异步通知的参数
	//    req：*http.Request
	//    返回参数 notifyReq：通知的参数
	//    返回参数 err：错误信息
	notifyReq, err := srv.ParseNotifyResult(req.Body) // c.Request()是 echo 框架的获取 *http.Request 的写法
	// 验签操作
	ok, err := wechat.VerifySign(srv.config["ApiKey"], wechat.SignType_MD5, notifyReq)
	if ok {
		rsp := new(wechat.NotifyResponse) // 回复微信的数据
		rsp.ReturnCode = gopay.SUCCESS
		rsp.ReturnMsg = gopay.OK
		xml = rsp.ToXmlString()
	} else {
		return xml, fmt.Errorf("Sign error: %s", err)
	}
	return xml, err
}

// ParseNotifyResult 解析异步通知
func (srv *Wechat) ParseNotifyResult(body string) (wxRsp *wechat.NotifyRequest, err error) {
	wxRsp = new(wechat.NotifyRequest)
	if err = xml.Unmarshal([]byte(body), wxRsp); err != nil {
		return wxRsp, fmt.Errorf("xml.Unmarshal(%s)：%w", string(body), err)
	}
	return wxRsp, err
}
