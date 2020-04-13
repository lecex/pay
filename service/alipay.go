package service

import (
	"fmt"

	proto "github.com/lecex/pay/proto/pay"
	"github.com/shopspring/decimal"

	"github.com/iGoogle-ink/gopay"
	"github.com/iGoogle-ink/gopay/alipay"
)

type Alipay struct {
	Client *alipay.Client
	config map[string]string
}

func (srv *Alipay) NewClient(config map[string]string, sandbox bool) {
	srv.config = config
	srv.Client = &alipay.Client{
		AppId:        config["AppId"],
		PrivateKey:   config["PrivateKey"],
		SignType:     config["SignType"],
		AppAuthToken: config["AppAuthToken"],
		// ReturnUrl:  config["ReturnUrl"],
		// NotifyUrl:  config["NotifyUrl"],
		IsProd: !sandbox,
	}
}

// AopF2F 商家扫用户付款码
//    文档地址：https://docs.open.alipay.com/api_1/alipay.trade.pay
func (srv *Alipay) AopF2F(order *proto.Order) (ok bool, err error) {
	body := make(gopay.BodyMap)
	body.Set("subject", order.Title)
	body.Set("scene", "bar_code")
	body.Set("auth_code", order.AuthCode)
	body.Set("out_trade_no", order.OrderNo)
	body.Set("total_amount", decimal.NewFromFloat(float64(order.TotalAmount)).Div(decimal.NewFromFloat(float64(100))))
	body.Set("timeout_express", "2m")
	body.Set("extend_params", map[string]interface{}{"sys_service_provider_id": srv.config["SysServiceProviderId"]})
	fmt.Println(body)
	aliRsp, err := srv.Client.TradePay(body)
	if err != nil {
		return ok, err
	}

	ok, err = alipay.VerifySyncSign(srv.config["AliPayPublicKey"], aliRsp.SignData, aliRsp.Sign)
	if err != nil {
		return ok, err
	}
	return ok, err
}
