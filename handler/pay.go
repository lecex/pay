package handler

import (
	"context"

	pd "github.com/lecex/pay/proto/pay"
	"github.com/lecex/pay/service"
	"github.com/lecex/pay/service/repository"
)

// Pay 支付结构
type Pay struct {
	Config repository.Config
	Order  repository.Order
	Alipay *service.Alipay
	Wechat *service.Wechat
}

// UserConfig 用户配置
func (srv *Pay) UserConfig(userId string) {

}

// CreateOrder 创建订单
func (srv *Pay) CreateOrder(req *pd.Order) {

}

// AopF2F 商家扫用户付款码
func (srv *Pay) AopF2F(ctx context.Context, req *pd.Request, res *pd.Response) (err error) {
	switch req.Order.Method {
	case "alipay":
		srv.Alipay.NewClient(map[string]string{
			"AppId":           "2016080900197401",
			"PrivateKey":      "MIIEpgIBAAKCAQEAqozLt6xArLodyKANcOEJTcPQiNkGZ1+/+GgCNTBYVJXR3m0CzWd5huMoZJ+YwsOPkhuh8/+BqAGjZPtNxtC1Tni9g4cqgXiIxdBaZwjd2UT0H7cMDiPBbo9g95gDLxSYNrpxqNwJPHn7NyCwyvzip66Pkp6K+fmYJHBXThYz06+y5SYtqOF1RCETa0rqmO23rYLlfbTMEXW25Rwb4p5GwoaKL2SLXEQflXcIcUmIXFgSj+MJiBrlTEe7ZKipAsOsjYjviu8semMHiVJOfuKPW8ciX24hNjpZeKuiklyu+w8WP8hxPUjVvTXGQrJlstLEUvljneUNz8jP/TDjiwUL1QIDAQABAoIBAQCKzHsex/j6mZ2ToW5O51YDC9GzDazAhJRfPYZOc0Hv1N4v/tfBAu1McaJ2Acz49N7rMcHkKZUDfhHUJRFvNHvZmTniySY2qDnng2GPaQ/jutJS3U3aVA8gQ1/PIM+2iTQ3lhTaL/j0VvG0M86t5JExlkcSNCU2u4KuZR8oVbloFMTo3tztCyH5nXv43/GqvJfS8GxMrf0o+bwHZ2HBiZ6Ea237caSixOhpt8ely3CEghciEXlWFQsxQkpxg7mBOH6xxkG0zW1ffthP8V0QGREbXS0C1FnATfrNFTDm25+jUGuaPFQOSjWqJ1Q35WyHYOr1oc3vT7bW5a7XdTSi4IQBAoGBAPZ6/80zhVW7oHO/dNvs4gJib6nC0z7dE1jrnclKqCwZ/LYnW1MV3PVfsnio8frFXArihc8QUjTj7avImdKgk7l2EqXM2rDGFNK/Gky9K3Nwiz3LwxXuy10LfrfW2z5qYhnFrVvR6ub3ZjuvVtXnSlTRrk2HB4yWPT7NJpTOt+51AoGBALEjDpSfZ80IBmoJeGRm8ALIZ8BLSQaMKuVwAw8pgyPYS9ebmgTDEvsj/RO640qzWSC7Lz/iWKz/k1VIqobz3CpV6Xcw7s3kF0JtU2YpIVnLmiiKmYM2WQHnFuLXZjkCGnbglzHIc8MhrGBhPYNekkrrLB7M4ijqdjqJczED2bvhAoGBAJcruyYc2kNJz1AOddrI/5kczIWe8zcUGmCoKd8iReC+k4sYul9MAngQGIL+g2MdlqUqZ40m4nSD8uowH3/acqAF9cvwx5Qx+OWExdmZEEQ+G3hsN5uFGP9ZJIAWa+NtFfvejMPLDLpZtD8Y/DY3JBS/gZsVHSExqCCTbH1KB+9dAoGBAIDpoX/aLsHRWFGtSLfRDlUIIjGY2LFyKvnFNgS/0lew3ykvbbyPd23cOB82wJmpwnCGqZFVmfF1InVLqAcEzDLnSTxcGT8wAxlt1OchgcsG2M8uZyBN2iL/WfGGjzdn53iiZIZveogFJp0Rx6GmntL1KavUsbbTQ23AgFuokLohAoGBAIAC7L7jJEDFUfk31CIH+xgrj0Y2RoPLgx86HM26TvflqVL/ym85QtzoP+4hrA4muUePUYFh9wDdUN7KrYKrRUXWDCPGIqB6bOL6a3rHcaodD6FUDnOHgSBmF/xJDE3K0BSqwMLkgpzxToofl1m6XZEe/TSDbXabIc5VnqSxswg0",
			"AliPayPublicKey": "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAruH561raPfR7mk3OFe/AKGsuir0oqfnwRekRtAo4EbliOlfSB2XAcQnyn6Wkc9bvRWkgGq6MJV2lVKRs114yyGz1MEhrjz8P1slp3KFnx/TwQgZSTGVH55BLNfB0cc+YA7/beTXHCOG4rQp8KPLURplkCMtuM/dQwS/6b/pF6dFHFhkZgXsHwtzK20jr6xVcT2Hk4tQGA1tfUSrskkj+CH61TSGfp5YkkfnieG3FEGfCjod0t37dCDKFNxD6EDOa10VqFtipLspo14PTDQmr3wQHCfZfmXqMdHtr2NMnIDYT4DCHhcUSI0VPMAohLbW6Y4Dm1JEkOyighLbrgY2qYQIDAQAB",
			"SignType":        "RSA2",
		}, true)
		ok, err := srv.Alipay.AopF2F(req.Order)
		if ok {
			res.Valid = ok
		}
		return err
	case "wechat":
		srv.Wechat.NewClient(map[string]string{
			"AppId":    "wxa4153f8d32d318f7",
			"MchId":    "1509529271",
			"ApiKey":   "4e47dc947158b67891381b6a14220d5f",
			"SubAppId": "wx48dc842d5284028c",
			"SubMchId": "1536451431",
		}, true)
		ok, err := srv.Wechat.AopF2F(req.Order)
		if ok {
			res.Valid = ok
		}
		return err
	}
	return err
}
