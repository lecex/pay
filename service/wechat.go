package service

import (
	"errors"
	"strconv"

	proto "github.com/lecex/pay/proto/pay"

	"github.com/bigrocs/wechat"
	"github.com/bigrocs/wechat/requests"
)

type Wechat struct {
	Client *wechat.Client
	config map[string]string
}

func (srv *Wechat) NewClient(config map[string]string, sandbox bool) {
	srv.config = config
	srv.Client = wechat.NewClient()
	c := srv.Client.Config
	c.AppId = config["AppId"]
	c.MchId = config["MchId"]
	c.ApiKey = config["ApiKey"]
	c.SubAppId = config["SubAppId"]
	c.SubMchId = config["SubMchId"]
}

// AopF2F 商家扫用户付款码
//    文档地址：https://pay.weixin.qq.com/wiki/doc/api/micropay_sl.php?chapter=9_10&index=1
func (srv *Wechat) AopF2F(order *proto.Order) (ok bool, err error) {
	// 配置参数
	request := requests.NewCommonRequest()
	request.Domain = "mch"
	request.ApiName = "pay.micropay"
	request.QueryParams = map[string]string{
		"auth_code":        order.AuthCode,
		"body":             order.Title,
		"out_trade_no":     order.OrderNo,
		"total_fee":        strconv.FormatInt(order.TotalAmount, 10),
		"spbill_create_ip": "127.0.0.1",
	}
	// 请求
	response, err := srv.Client.ProcessCommonRequest(request)
	if err != nil {
		return ok, err
	}
	req, err := response.GetHttpContentMap()
	if err != nil {
		return ok, err
	}
	if req["result_code"] == "SUCCESS" {
		return true, err
	}
	return ok, errors.New(response.GetHttpContentJson())
}
