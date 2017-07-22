package http

type Header interface {
	Add(key, value string)
	Set(key, value string)
	Get(key string) string
}
