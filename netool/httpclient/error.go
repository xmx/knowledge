package httpclient

import "fmt"

type Error struct {
	Code int
	Text string
}

func (e *Error) Error() string {
	return fmt.Sprintf("http response status %d, message is: %s", e.Code, e.Text)
}
