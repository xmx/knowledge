package httpclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	cli *http.Client
}

// New 创建 httpclient
func New(cli ...*http.Client) *Client {
	if len(cli) != 0 {
		return &Client{cli: cli[0]}
	}

	hc := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives:   true,
			DisableCompression:  true,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxIdleConns:        10,
			MaxConnsPerHost:     10,
			MaxIdleConnsPerHost: 10,
		},
	}

	return &Client{cli: hc}
}

// Get 发送 GET 请求
func (hc Client) Get(addr string, queries url.Values, opts ...Option) (io.ReadCloser, error) {
	return hc.exec(http.MethodGet, addr, queries, nil, opts...)
}

// GetJSON 发送 GET 请求，返回数据为 JSON
func (hc Client) GetJSON(addr string, queries url.Values, reply any, opts ...Option) error {
	opt := WithHeader("Accept", "application/json")
	opts = append(opts, opt)

	rc, err := hc.exec(http.MethodGet, addr, queries, nil, opts...)
	if err != nil || rc == nil { // 请求失败或请求成功但是 Body 为 nil，比如：http.StatusNoContent
		return err
	}

	err = json.NewDecoder(rc).Decode(reply)
	_ = rc.Close()

	return err
}

// Post 发送 POST 请求
func (hc Client) Post(addr string, queries url.Values, body io.Reader, opts ...Option) (io.ReadCloser, error) {
	return hc.exec(http.MethodPost, addr, queries, body, opts...)
}

// PostForm 发送 POST 请求，数据 body 为 FormData 格式
func (hc Client) PostForm(addr string, queries url.Values, body url.Values, opts ...Option) (io.ReadCloser, error) {
	opt := WithHeader("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	opts = append(opts, opt)

	return hc.Post(addr, queries, strings.NewReader(body.Encode()), opts...)
}

// PostJSON 发送 POST 请求，请求格式为 JSON，返回数据格式为 JSON
func (hc Client) PostJSON(addr string, queries url.Values, body, reply any, opts ...Option) error {
	buf := new(bytes.Buffer)
	if body != nil {
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return err
		}
	}

	opt := WithHeader("Content-Type", "application/json; charset=utf-8")
	opts = append(opts, opt)

	rc, err := hc.Post(addr, queries, buf, opts...)
	if err != nil || rc == nil { // 请求失败或请求成功但是 Body 为 nil，比如：http.StatusNoContent
		return err
	}

	err = json.NewDecoder(rc).Decode(reply)
	_ = rc.Close()

	return err
}

// Put 发送 HTTP PUT 请求
func (hc Client) Put(addr string, queries url.Values, body io.Reader, opts ...Option) (io.ReadCloser, error) {
	return hc.exec(http.MethodPut, addr, queries, body, opts...)
}

// exec 执行发送逻辑
func (hc Client) exec(method, addr string, queries url.Values, body io.Reader, opts ...Option) (rc io.ReadCloser, err error) {
	opt := &option{header: make(http.Header, 8)}
	for _, fn := range opts {
		fn(opt)
	}
	if opt.timeout <= 0 {
		opt.timeout = 5 * time.Second
	}
	if opt.retry > 0 && opt.delay <= 0 {
		opt.delay = time.Second
	}

	if len(queries) != 0 {
		if addr, err = hc.appendQueries(addr, queries); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, addr, body)
	if err != nil {
		return nil, err
	}
	if opt.host != "" {
		req.Host = opt.host
	}
	req.Header = opt.header

	rc, err = hc.fetch(req)
	if err == nil || opt.retry <= 0 || !hc.canRetry(err) {
		return rc, err
	}

	for i := 0; i < opt.retry; i++ {
		time.Sleep(opt.delay)
		if rc, err = hc.fetch(req); err == nil || !hc.canRetry(err) {
			break
		}
	}

	return
}

// fetch 发送 http 请求
func (hc Client) fetch(req *http.Request) (io.ReadCloser, error) {
	res, err := hc.cli.Do(req)
	if err != nil {
		return nil, err
	}

	code := res.StatusCode
	if code >= http.StatusOK && code < http.StatusMultipleChoices {
		return res.Body, nil
	}

	txt := make([]byte, 1024)
	n, _ := io.ReadFull(res.Body, txt)
	_ = res.Body.Close()
	err = &Error{Code: code, Text: string(txt[:n])}

	return nil, err
}

// appendQueries 将参数合并
// 	example:
//			addr: https://18.com.cn/?name=jack
// 			queries: map[age][]string{"18"}
// 		合并后: https://18.com.cn/?name=jack&age=18
func (hc Client) appendQueries(addr string, queries url.Values) (string, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return "", err
	}
	if u.RawQuery != "" {
		values, ex := url.ParseQuery(u.RawQuery)
		if ex != nil {
			return "", ex
		}
		for key, vals := range values {
			for _, val := range vals {
				queries.Add(key, val)
			}
		}
	}

	u.RawQuery = queries.Encode()

	return u.String(), nil
}

// retry 判断是否需要重试请求
func (Client) canRetry(err error) bool {
	switch e := err.(type) {
	case nil:
		return false
	case *Error:
		return e.Code >= http.StatusInternalServerError
	default:
		return true
	}
}
