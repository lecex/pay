# TradeAPI 支付接口文档 proto

```
    APIURL
    https://xxx.com/pay-api/trades/AopF2F
    https://xxx.com/pay-api/trades/Query
    https://xxx.com/pay-api/trades/Refund
    https://xxx.com/pay-api/trades/RefundQuery
```

```
syntax = "proto3";

package trade;

service Trades {
    // AopF2F 商家扫用户付款码
    rpc AopF2F(Request) returns (Response) {}
    rpc Query(Request) returns (Response) {}
    rpc Refund(Request) returns (Response) {} // 退款接口
    rpc RefundQuery(Request) returns (Response) {} // 退款查询接口
}
// 请求参数
message BizContent {
    string auth_code = 1;       // 付款码                 必选接口[AopF2F]                              280528574232947539
    string title = 2;           // 订单标题               必选接口[AopF2F]                              测试商品名称
    int64 total_fee = 3;        // 订单总金额 [单位分]     必选接口[AopF2F]                              180=1.8元
    string out_trade_no = 4;    // 订单编号               必选接口[AopF2F、Query、Refund、RefundQuery]   GZ2020010117534314525
    int64 refund_fee = 5;       // 退款金额 [单位分]       必选接口[Refund]                              180=1.8元
    string operator_id = 6;     // 商户操作员编号          必选接口[]                                    1004
    string terminal_id = 7;     // 商户机具终端编号        必选接口[]                                    6008
    string attach = 8;          // 附加数据 [数据格式json] 必选接口[]                                    {data:data}
}
// 公共请求参数
message Request {
    string app_id = 1;          // 应用APPID                                        必选
    string channel = 2;         // 通道名称 [支付宝、微信、银联等]                      必选
    string sign_type = 3;       // 商户生成签名字符串所使用的签名算法类型，目前支持RSA2    必选
    string sign = 4;            // 商户请求参数的签名串，详见签名                       必选
    BizContent biz_content = 5; // 请求内容                                         必选

}
// 退款订单
message Refund {
    string out_trade_no = 1;            // 订单编号         必选接口[Refund、RefundQuery]     GZ2020010117534314525
    string total_fee = 2;               // 订单金额 [单位分] 必选接口[Refund、RefundQuery]     180=1.8元
    string refund_fee = 3;              // 退款金额 [单位分] 必选接口[Refund、RefundQuery]     180=1.8元
    string status = 4;                  // 订单状态         必选接口[Refund、RefundQuery]     SUCCESS成功、CLOSED关闭、USERPAYING等待用户付款、WAIT系统繁忙稍后查询
}
// 响应参数
message Content {
    string return_code = 1;             // 业务结果         必选接口[AopF2F、Query、Refund、RefundQuery]     SUCCESS/FAIL
    string return_msg = 2;              // 返回消息         必选接口[AopF2F、Query、Refund、RefundQuery]     支付失败
    string channel = 3;                 // 通道内容         必选接口[AopF2F、Query、Refund、RefundQuery]     wechat
    string out_trade_no = 4;            // 订单编号         必选接口[AopF2F、Query、Refund、RefundQuery]     GZ2020010117534314525
    string trade_no = 5;                // 渠道交易编号      必选接口[AopF2F、Query、Refund、RefundQuery]     2013112011001004330000121536
    string total_fee = 6;               // 订单金额 [单位分] 必选接口[AopF2F、Query、Refund、RefundQuery]     180=1.8元
    string refund_fee = 7;              // 退款金额 [单位分] 必选接口[Refund、RefundQuery]                   180=1.8元
    string status = 8;                  // 订单状态         必选接口[AopF2F、Query、Refund、RefundQuery]     SUCCESS成功、CLOSED关闭、USERPAYING等待用户付款、WAIT系统繁忙稍后查询
    string time_end = 9;                // 订单完成时间      必选接口[AopF2F、Query、Refund、RefundQuery]     SUCCESS成功、CLOSED关闭、USERPAYING等待用户付款、WAIT系统繁忙稍后查询
    string wechat_open_id = 10;         // 微信openid       必选接口[]                                      oUpF8uN95-Pteags6E_roPHg7AG
    string wechat_is_subscribe = 11;    // 是否微信关注公众号 必选接口[]                                      Y/N
    string alipay_buyer_logon_id = 12;  // 支付宝账号        必选接口[]                                     158****1562
    string alipay_buyer_user_id = 13;   // 支付宝用户id      必选接口[]                                     2088101117955611
    repeated Refund refund_list = 14;   // 退款订单明细       必选接口[Refund、RefundQuery]
}
// 公共响应参数
message Response {
    string sign = 1;            // 签名 [计算content得到的秘钥签名]           必选
    Content content = 2;        // 响应参数合计                             必选
}

```
