package service

import (
	"encoding/json"
	"fmt"

	notifyPB "github.com/lecex/pay/proto/notify"
	proto "github.com/lecex/pay/proto/pay"
	"github.com/lecex/pay/uitl"
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
	body.Set("timeout_express", "30m")
	body.Set("extend_params", map[string]interface{}{"sys_service_provider_id": srv.config["SysServiceProviderId"]})

	aliRsp, err := srv.Client.TradePay(body)
	if err != nil {
		return ok, err
	}

	return alipay.VerifySyncSign(srv.config["AliPayPublicKey"], aliRsp.SignData, aliRsp.Sign)
}

// Notify 异步通知
//    文档地址：https://opendocs.alipay.com/open/200/106120
func (srv *Alipay) Notify(req *notifyPB.Request) (ok bool, err error) {
	// ====异步通知参数解析和验签Sign====
	// 解析异步通知的参数
	//    req：*http.Request
	//    返回参数 notifyReq：通知的参数
	//    返回参数 err：错误信息
	notifyReq, err := srv.parseNotifyResult(req) // c.Request()是 echo 框架的获取
	// 验签操作
	return alipay.VerifySign(srv.config["AliPayPublicKey"], notifyReq)
}

// ParseNotifyResult 解析支付宝支付异步通知的参数到Struct
//    req：*http.Request
//    返回参数notifyReq：Notify请求的参数
//    返回参数err：错误信息
//    文档：https://docs.open.alipay.com/203/105286/
func (srv *Alipay) parseNotifyResult(req *notifyPB.Request) (notifyReq *alipay.NotifyRequest, err error) {
	notifyReq = new(alipay.NotifyRequest)
	notifyReq.NotifyTime = uitl.Get(req, "notify_time")
	notifyReq.NotifyType = uitl.Get(req, "notify_type")
	notifyReq.NotifyId = uitl.Get(req, "notify_id")
	notifyReq.AppId = uitl.Get(req, "app_id")
	notifyReq.Charset = uitl.Get(req, "charset")
	notifyReq.Version = uitl.Get(req, "version")
	notifyReq.SignType = uitl.Get(req, "sign_type")
	notifyReq.Sign = uitl.Get(req, "sign")
	notifyReq.AuthAppId = uitl.Get(req, "auth_app_id")
	notifyReq.TradeNo = uitl.Get(req, "trade_no")
	notifyReq.OutTradeNo = uitl.Get(req, "out_trade_no")
	notifyReq.OutBizNo = uitl.Get(req, "out_biz_no")
	notifyReq.BuyerId = uitl.Get(req, "buyer_id")
	notifyReq.BuyerLogonId = uitl.Get(req, "buyer_logon_id")
	notifyReq.SellerId = uitl.Get(req, "seller_id")
	notifyReq.SellerEmail = uitl.Get(req, "seller_email")
	notifyReq.TradeStatus = uitl.Get(req, "trade_status")
	notifyReq.TotalAmount = uitl.Get(req, "total_amount")
	notifyReq.ReceiptAmount = uitl.Get(req, "receipt_amount")
	notifyReq.InvoiceAmount = uitl.Get(req, "invoice_amount")
	notifyReq.BuyerPayAmount = uitl.Get(req, "buyer_pay_amount")
	notifyReq.PointAmount = uitl.Get(req, "point_amount")
	notifyReq.RefundFee = uitl.Get(req, "refund_fee")
	notifyReq.Subject = uitl.Get(req, "subject")
	notifyReq.Body = uitl.Get(req, "body")
	notifyReq.GmtCreate = uitl.Get(req, "gmt_create")
	notifyReq.GmtPayment = uitl.Get(req, "gmt_payment")
	notifyReq.GmtRefund = uitl.Get(req, "gmt_refund")
	notifyReq.GmtClose = uitl.Get(req, "gmt_close")
	notifyReq.PassbackParams = uitl.Get(req, "passback_params")

	billList := uitl.Get(req, "fund_bill_list")
	if billList != gopay.NULL {
		bills := make([]*alipay.FundBillListInfo, 0)
		if err = json.Unmarshal([]byte(billList), &bills); err != nil {
			return nil, fmt.Errorf(`"fund_bill_list" xml.Unmarshal(%s)：%w`, billList, err)
		}
		notifyReq.FundBillList = bills
	} else {
		notifyReq.FundBillList = nil
	}

	detailList := uitl.Get(req, "voucher_detail_list")
	if detailList != gopay.NULL {
		details := make([]*alipay.VoucherDetailListInfo, 0)
		if err = json.Unmarshal([]byte(detailList), &details); err != nil {
			return nil, fmt.Errorf(`"voucher_detail_list" xml.Unmarshal(%s)：%w`, detailList, err)
		}
		notifyReq.VoucherDetailList = details
	} else {
		notifyReq.VoucherDetailList = nil
	}
	return
}
