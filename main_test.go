package main

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	configPB "github.com/lecex/pay/proto/config"
	payPB "github.com/lecex/pay/proto/pay"
	db "github.com/lecex/pay/providers/database"
	"github.com/lecex/pay/service"
	"github.com/lecex/pay/service/repository"

	"github.com/lecex/pay/handler"
)

func TestConfigSelfUpdate(t *testing.T) {
	req := &configPB.Request{
		Config: &configPB.Config{
			Id: `bvbv011511212`,
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
	err := h.SelfUpdate(context.TODO(), req, res)
	fmt.Println("ConfigGet", res, err)
	t.Log(req, res, err)
}

func TestConfigGet(t *testing.T) {
	req := &configPB.Request{
		Config: &configPB.Config{
			Id: `bvbv011511211`,
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
	// fmt.Println("ConfigGet", res, err)
	t.Log(req, res, err)
}

func TestAopF2FWechat(t *testing.T) {
	h := handler.Pay{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		Alipay: &service.Alipay{},
		Wechat: &service.Wechat{},
	}
	req := &payPB.Request{
		Order: &payPB.Order{
			StoreName:   `bvbv01`,
			Method:      `wechat`,
			AuthCode:    `134711673107134820`,
			Title:       `Wechat扫码支付`,
			OrderNo:     `GTZ2020010117534314591`,
			TotalAmount: 2,
		},
	}
	res := &payPB.Response{}
	err := h.AopF2F(context.TODO(), req, res)
	// fmt.Println(req, res, err)
	t.Log("TestAopF2FWechat", req, res, err)

}

func TestCancel(t *testing.T) {
	h := handler.Pay{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		Alipay: &service.Alipay{},
		Wechat: &service.Wechat{},
	}
	req := &payPB.Request{
		Order: &payPB.Order{
			StoreName: `bvbv01`,
			OrderNo:   `GTZ2020010117534314591`,
		},
	}
	res := &payPB.Response{}
	err := h.Cancel(context.TODO(), req, res)
	// fmt.Println(req, res, err)
	t.Log("TestRefundWechat", req, res, err)

}

func TestAffirmRefund(t *testing.T) {
	h := handler.Pay{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		Alipay: &service.Alipay{},
		Wechat: &service.Wechat{},
	}
	req := &payPB.Request{
		Order: &payPB.Order{
			StoreName: `bvbv01`,
			OrderNo:   `GTZ2020010117534314591_010101`,
		},
	}
	res := &payPB.Response{}
	err := h.AffirmRefund(context.TODO(), req, res)
	// fmt.Println(req, res, err)
	t.Log("TestAffirmRefund", req, res, err)

}

func TestDD(t *testing.T) {
	alipay := "283971647685282846"
	wechat := "136731678237198135"
	fmt.Println(regexp.Match(`^(?:2[5-9]|30)\d{14,18}$`, []byte(alipay)))
	fmt.Println(regexp.Match(`^1[0-5]\d{16}$`, []byte(wechat)))
	fmt.Println(456, alipay, wechat)

}
