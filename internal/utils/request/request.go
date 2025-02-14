package request

import (
	"net/http"

	"github.com/go-resty/resty/v2"
)

var Request = resty.New().SetTransport(&http.Transport{
	Proxy: http.ProxyFromEnvironment, // 通用适配环境变量
}).SetRetryCount(3)
