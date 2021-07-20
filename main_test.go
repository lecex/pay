package main

import (
	"context"
	"fmt"
	"os"
	"testing"

	configPB "github.com/lecex/pay/proto/config"
	tradePB "github.com/lecex/pay/proto/trade"
	db "github.com/lecex/pay/providers/database"
	"github.com/lecex/pay/service/repository"
	"github.com/lecex/pay/service/trade"

	"github.com/lecex/pay/handler"
)

func TestConfigSelfUpdate(t *testing.T) {
	req := &configPB.Request{
		Config: &configPB.Config{
			Id:        `7b490bb0-c04d-4fd8-9bf9-ef4f2239d3a0`,
			StoreName: "ceshi1",
			// Channel:   "icbc",
			Status: true,
			Alipay: &configPB.Alipay{
				AppAuthToken: os.Getenv("PAY_ALIPAY_APP_AUTH_TOKEN"),
				Fee:          38,
			},
			Wechat: &configPB.Wechat{
				SubMchId: os.Getenv("PAY_WECHAT_SUB_MCH_ID"),
				Fee:      38,
			},
			Icbc: &configPB.Icbc{
				SubMerId: os.Getenv("PAY_ICBC_SUB_MER_ID"),
				Fee:      0,
			},
		},
	}
	res := &configPB.Response{}
	handler := &handler.Handler{}
	h := handler.Config()
	err := h.SelfUpdate(context.TODO(), req, res)
	// fmt.Println("ConfigSelfUpdate", req, res, err)
	t.Log(req, res, err)
}

func TestConfigGet(t *testing.T) {
	req := &configPB.Request{
		Config: &configPB.Config{
			StoreName: `ceshi`,
		},
	}
	res := &configPB.Response{}
	handler := &handler.Handler{}
	h := handler.Config()
	err := h.Get(context.TODO(), req, res)
	// fmt.Println("ConfigGet", res, err)
	t.Log(req, res, err)
}

func TestAopF2FWechat(t *testing.T) {
	h := handler.Trade{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		Alipay: &trade.Alipay{},
		Wechat: &trade.Wechat{},
		Icbc:   &trade.Icbc{},
	}
	req := &tradePB.Request{
		StoreId: "7b490bb0-c04d-4fd8-9bf9-ef4f2239d3a0",
		BizContent: &tradePB.BizContent{
			Channel:    "alipay",
			AuthCode:   `287359201063620602`,
			Title:      `IcbcAlipay扫码支付`,
			OutTradeNo: `GTB202001011753431459029`,
			TotalFee:   2,
			OperatorId: "0001",
			TerminalId: "9008",
			Attach:     `{"code": "001"}`,
		},
	}
	res := &tradePB.Response{}
	err := h.AopF2F(context.TODO(), req, res)
	// fmt.Println("TestAopF2FWechat____________", res, err)
	t.Log("TestAopF2FWechat", req, res, err)

}

func TestQuery(t *testing.T) {
	h := handler.Trade{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		Alipay: &trade.Alipay{},
		Wechat: &trade.Wechat{},
		Icbc:   &trade.Icbc{},
	}
	req := &tradePB.Request{
		StoreId: "7b490bb0-c04d-4fd8-9bf9-ef4f2239d3a0",
		BizContent: &tradePB.BizContent{
			Channel:    "alipay",
			OutTradeNo: `GTB202001011753431459029`,
		},
	}
	res := &tradePB.Response{}
	err := h.Query(context.TODO(), req, res)
	// fmt.Println("TestQuery___", res, err)
	t.Log("TestQuery", req, res, err)

}

func TestRefund(t *testing.T) {
	h := handler.Trade{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		Alipay: &trade.Alipay{},
		Wechat: &trade.Wechat{},
		Icbc:   &trade.Icbc{},
	}
	req := &tradePB.Request{
		StoreId: "7b490bb0-c04d-4fd8-9bf9-ef4f2239d3a0",
		BizContent: &tradePB.BizContent{
			OutTradeNo:  `GTB2020010117534314459033`,
			OutRefundNo: `GTB2020010117534314459033_101013`,
		},
	}
	res := &tradePB.Response{}
	err := h.Refund(context.TODO(), req, res)
	// fmt.Println("TestRefund_____", res, err)
	t.Log("TestAffirmRefund", req, res, err)

}

func TestRefundQuery(t *testing.T) {
	h := handler.Trade{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		Alipay: &trade.Alipay{},
		Wechat: &trade.Wechat{},
		Icbc:   &trade.Icbc{},
	}
	req := &tradePB.Request{
		StoreId: "7b490bb0-c04d-4fd8-9bf9-ef4f2239d3a0",
		BizContent: &tradePB.BizContent{
			OutTradeNo:  `GTB202001011753431459033`,
			OutRefundNo: `GTB202001011753431459033_101013`,
		},
	}
	res := &tradePB.Response{}
	err := h.RefundQuery(context.TODO(), req, res)
	fmt.Println("TestRefundQuery_____", res, err)
	t.Log("TestAffirmRefund", req, res, err)

}
