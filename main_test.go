package main

import (
	"context"
	"fmt"
	"testing"

	configPB "github.com/lecex/pay/proto/config"

	"github.com/lecex/pay/handler"
)

func TestConfigGet(t *testing.T) {
	req := &configPB.Request{
		Config: &configPB.Config{
			Id: `bvbv01151121`,
			Alipay: &configPB.Alipay{
				AppId:           `saasas`,
				PrivateKey:      `asdfqwwqqw`,
				AliPayPublicKey: `asasq`,
				SignType:        `asasas`,
			},
			Wechat: &configPB.Wechat{
				AppId:    `qwwxzas`,
				MchId:    `aswsqwqw`,
				ApiKey:   `aq121212`,
				SubAppId: `asasqwqw`,
				SubMchId: `swqqwqwqw`,
			},
		},
	}
	res := &configPB.Response{}
	handler := &handler.Handler{}
	h := handler.Config()
	err := h.Create(context.TODO(), req, res)
	fmt.Println("ConfigGet", res, err)
	t.Log(req, res, err)
}

// func TestAopF2FAlipay(t *testing.T) {
// 	alipay := &service.Alipay{}
// 	h := handler.Pay{nil, nil, alipay, nil}
// 	req := &payPB.Request{
// 		Order: &payPB.Order{
// 			Method:      `alipay`,
// 			AuthCode:    `285653237303565644`,
// 			Title:       `Alipay扫码支付`,
// 			OrderSn:     `GZ202001011753431451`,
// 			TotalAmount: 200001,
// 		},
// 	}
// 	res := &payPB.Response{}
// 	err := h.AopF2F(context.TODO(), req, res)
// 	// fmt.Println(req, res, err)
// 	t.Log(req, res, err)

// }

// func TestAopF2FWechat(t *testing.T) {
// 	wecaht := &service.Wechat{}
// 	h := handler.Pay{nil, nil, nil, wecaht}
// 	req := &payPB.Request{
// 		Order: &payPB.Order{
// 			Method:      `wechat`,
// 			AuthCode:    `134527825438234112`,
// 			Title:       `Wechat扫码支付`,
// 			OrderSn:     `GZ202001011753431461`,
// 			TotalAmount: 1,
// 		},
// 	}
// 	res := &payPB.Response{}
// 	err := h.AopF2F(context.TODO(), req, res)
// 	// fmt.Println(req, res, err)
// 	t.Log(req, res, err)

// }
