package codec

import "io"

type Header struct {
	ServiceMethod string
	Seq           uint64
	Error         string
}

type ICodec interface {
	io.Closer
	ReadHeader(h *Header) error
	ReadBody(b interface{}) error
	Write(*Header, interface{}) error
}

type CodecType string
const (
	GobType  = "application/gob"
	JsonType = "application/json"
)

type NewCodecFunc func(io.ReadWriteCloser) ICodec

var NewCodecFuncMap map[CodecType]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[CodecType]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}