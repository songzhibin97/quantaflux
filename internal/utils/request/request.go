package request

import "github.com/go-resty/resty/v2"

var Request = resty.New().SetRetryCount(3)
