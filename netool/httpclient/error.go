package httpclient

import "fmt"

type Error struct {
	Code int    // HTTP 响应状态码
	Text string // 出现错误时返回的 Body，取前 1024 个 byte
}

func (e *Error) Error() string {
	return fmt.Sprintf("http response status %d, message is: %s", e.Code, e.Text)
}
