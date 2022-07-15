package httpclient

import (
	"net/http"
	"time"
)

type option struct {
	header  http.Header
	timeout time.Duration
	retry   int
	delay   time.Duration
	host    string
}

type Option func(o *option)

func WithHeader(key, val string) Option {
	return func(o *option) {
		o.header.Add(key, val)
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(o *option) {
		o.timeout = timeout
	}
}

func WithRetry(n int) Option {
	return func(o *option) {
		o.retry = n
	}
}

func WithDelay(delay time.Duration) Option {
	return func(o *option) {
		o.delay = delay
	}
}

func WithHost(host string) Option {
	return func(o *option) {
		o.host = host
	}
}
