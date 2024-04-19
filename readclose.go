package gptscript

import "io"

// reader is a dummy io.Reader that returns EOF. This is used in situations where errors are returned to allow
// code to always call Read without having to check for nil.
type reader struct{}

func (r reader) Read([]byte) (int, error) {
	return 0, io.EOF
}
