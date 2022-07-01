package codec

import "io"

type Header struct {
	ServiceMethod string
	Seq           uint64
	Error         error
}

type ICodec interface {
	io.Closer
	ReadHeader(h *Header) error
	ReadBody(b interface{}) error
	Write(*Header, interface{}) error
}