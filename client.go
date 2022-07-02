package geerpc

import (
	"encoding/json"
	"geerpc/codec"
	"net"
)

type Client struct {
	cc codec.ICodec
}

func Dial(network, addr string) (*Client, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	// negotiate on Option
	_ = json.NewEncoder(conn).Encode(DefaultOption)

	// initialize codec
	cc := codec.NewCodecFuncMap[DefaultOption.CodecType](conn)
	return NewClient(cc), nil
}

func NewClient(cc codec.ICodec) *Client {
	return &Client{
		cc,
	}
}

func (c *Client) Call(ServiceMethod string, argv, reply interface{}) error {
	return nil
}

// keep receiving from server
func (c *Client) receive() {

}

type Call struct {
	ServiceMethod string
	Seq           uint64
	Error         string
	Args, Reply   interface{}
}
