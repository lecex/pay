package uitl

import (
	"strings"

	pb "github.com/lecex/pay/proto/notify"
)

// Get 获取请求值
func Get(req *pb.Request, key string) (value string) {
	// parse values from the get request
	v, ok := req.Get[key]
	if !ok || len(v.Values) == 0 {
		return value
	}
	value = strings.Join(v.Values, " ")
	return value
}
