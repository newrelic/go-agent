package http

import (
	"net/url"
)

type Request interface {
	URL() *url.URL
	Method() string
	Header() Header
}
