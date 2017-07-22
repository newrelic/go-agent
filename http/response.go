package http

type Response interface {
	Header() Header
	Code() int
	Request() Request
}
