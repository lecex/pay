package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/clbanning/mxj"

	configPB "github.com/lecex/pay/proto/config"
	pb "github.com/lecex/pay/proto/notify"
	orderPB "github.com/lecex/pay/proto/order"
	"github.com/lecex/pay/service"
	"github.com/lecex/pay/service/repository"
	"github.com/lecex/pay/util"
)

// Notify 订单
type Notify struct {
	Config repository.Config
	Repo   repository.Order
	alipay *service.Alipay
	wechat *service.Wechat
}

// UserConfig 用户配置
func (srv *Notify) UserConfig(order *orderPB.Order) (*configPB.Config, error) {
	config := &configPB.Config{}
	if order.StoreId != "" {
		config.Id = order.StoreId
	}
	err := srv.Config.Get(config)
	return config, err
}

// Alipay 异步通知
func (srv *Notify) Alipay(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	OrderNo := util.Get(req, "out_trade_no")
	if OrderNo == "" {
		return fmt.Errorf("out_trade_no no content")
	}
	order := &orderPB.Order{
		OrderNo: util.Get(req, "out_trade_no"), // 订单编号
	}
	if err = srv.Repo.Get(order); err != nil {
		return err
	}
	if order.Method == "" {
		return fmt.Errorf("Method:%s no content", order.Method)
	}
	config, err := srv.UserConfig(order)
	if err != nil {
		return fmt.Errorf("StoreId :%s config not found ", order.StoreId)
	}
	if order.Method == "alipay" {
		srv.alipay.NewClient(map[string]string{
			"AppId":                config.Alipay.AppId,
			"PrivateKey":           config.Alipay.PrivateKey,
			"AliPayPublicKey":      config.Alipay.AliPayPublicKey,
			"AppAuthToken":         config.Alipay.AppAuthToken,
			"SysServiceProviderId": config.Alipay.SysServiceProviderId,
			"SignType":             config.Alipay.SignType,
		}, config.Alipay.Sandbox)
		ok, err := srv.alipay.Notify(req)
		if ok {
			order.Stauts = 1
			err = srv.Repo.Update(order)
			if err != nil {
				return err
			}
			res.StatusCode = http.StatusOK
			res.Body = string("success")
		} else {
			return fmt.Errorf("Verify Sign sgin error:  ", err)
		}
	} else {
		return fmt.Errorf("Method: %s not alipay ", order.Method)
	}

	return err
}

// Wechat 异步通知
func (srv *Notify) Wechat(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	if req.Body == "" {
		return fmt.Errorf("Body Empty")
	}
	wxRsp, err := srv.wechat.ParseNotifyResult(req.Body)
	if err != nil {
		return err
	}
	order := &orderPB.Order{
		OrderNo: wxRsp["out_trade_no"].(string), // 订单编号
	}
	if err = srv.Repo.Get(order); err != nil {
		return err
	}
	if order.Method == "" {
		return fmt.Errorf("Method:%s no content", order.Method)
	}
	config, err := srv.UserConfig(order)
	if err != nil {
		return fmt.Errorf("StoreId :%s config not found ", order.StoreId)
	}
	if order.Method == "wechat" {
		srv.wechat.NewClient(map[string]string{
			"AppId":    config.Wechat.AppId,
			"MchId":    config.Wechat.MchId,
			"ApiKey":   config.Wechat.ApiKey,
			"SubAppId": config.Wechat.SubAppId,
			"SubMchId": config.Wechat.SubMchId,
		}, config.Wechat.Sandbox)
		ok, err := srv.wechat.Notify(req.Body)
		if ok {
			order.Stauts = 1
			err = srv.Repo.Update(order)
			if err != nil {
				return err
			}
			vm := mxj.Map{}
			vm["return_code"] = "SUCCESS"
			vm["return_msg"] = "OK"
			body, _ := vm.Xml()
			res.StatusCode = http.StatusOK
			res.Body = string(body)
		} else {
			return fmt.Errorf("Verify Sign sgin error:  ", err)
		}
	} else {
		return fmt.Errorf("Method: %s not wechat ", order.Method)
	}

	return err
}
