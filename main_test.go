package main

import (
	"context"
	"fmt"
	"os"
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
			Id:        `bvbv011511212`,
			StoreName: "ceshi",
			Channel:   "icbc",
			Stauts:    true,
			Alipay: &configPB.Alipay{
				AppId:                os.Getenv("ALIPAY_APPID"),
				PrivateKey:           os.Getenv("ALIPAY_PRIVATE_KEY"),
				AliPayPublicKey:      os.Getenv("ALIPAY_PUBLIC_KEY"),
				SignType:             "RSA2",
				AppAuthToken:         os.Getenv("ALIPAY_APP_AUTH_TOKEN"),
				SysServiceProviderId: os.Getenv("ALIPAY_SYS_SERVICE_PROVIDER_ID"),
				Fee:                  38,
			},
			Wechat: &configPB.Wechat{
				AppId:    os.Getenv("WECHAT_APPID"),
				MchId:    os.Getenv("WECHAT_MCH_ID"),
				ApiKey:   os.Getenv("WECHAT_API_KEY"),
				PemCert:  os.Getenv("WECHAT_PEM_CERT"),
				PemKey:   os.Getenv("WECHAT_PEM_KEY"),
				SubAppId: os.Getenv("WECHAT_SUB_APP_ID"),
				SubMchId: os.Getenv("WECHAT_SUB_MCH_ID"),
				Fee:      38,
			},
			Icbc: &configPB.Icbc{
				AppId:          os.Getenv("ICBC_APPID"),
				PrivateKey:     os.Getenv("ICBC_PRIVATE_KEY"),
				IcbcPublicKey:  os.Getenv("ICBC_PUBLIC_KEY"),
				SignType:       "RSA2",
				ReturnSignType: "RSA",
				MerId:          os.Getenv("ICBC_MER_ID"),
				Fee:            38,
			},
		},
	}
	res := &configPB.Response{}
	handler := &handler.Handler{}
	h := handler.Config()
	err := h.SelfUpdate(context.TODO(), req, res)
	// fmt.Println("ConfigSelfUpdate", res, err)
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
	h := handler.Pay{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		Alipay: &service.Alipay{},
		Wechat: &service.Wechat{},
		Icbc:   &service.Icbc{},
	}
	req := &payPB.Request{
		Order: &payPB.Order{
			StoreName:   `ceshi`,
			Method:      `icbc`,
			AuthCode:    `286203470917515029`,
			Title:       `IcbcAlipay扫码支付`,
			OrderNo:     `GTZ202001011753431459022`,
			TotalAmount: 1,
		},
	}
	res := &payPB.Response{}
	err := h.AopF2F(context.TODO(), req, res)
	// fmt.Println("____________A", res, err)
	t.Log("TestAopF2FWechat", req, res, err)

}

func TestQuery(t *testing.T) {
	h := handler.Pay{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		Alipay: &service.Alipay{},
		Wechat: &service.Wechat{},
		Icbc:   &service.Icbc{},
	}
	req := &payPB.Request{
		Order: &payPB.Order{
			StoreName: `ceshi`,
			OrderNo:   `GTZ202001011753431459022`,
		},
	}
	res := &payPB.Response{}
	err := h.Query(context.TODO(), req, res)
	// fmt.Println("TestQuery___", res, err)
	t.Log("TestQuery", req, res, err)

}

func TestRefund(t *testing.T) {
	h := handler.Pay{
		Config: &repository.ConfigRepository{db.DB},
		Repo:   &repository.OrderRepository{db.DB},
		Alipay: &service.Alipay{},
		Wechat: &service.Wechat{},
		Icbc:   &service.Icbc{},
	}
	req := &payPB.Request{
		Order: &payPB.Order{
			StoreName:       `ceshi`,
			OriginalOrderNo: `GTZ202001011753431459022`,
		},
	}
	res := &payPB.Response{}
	err := h.Refund(context.TODO(), req, res)
	fmt.Println("TestRefund_____", req, res, err)
	t.Log("TestAffirmRefund", req, res, err)

}
