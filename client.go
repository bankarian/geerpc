package geerpc

import (
	"encoding/json"
	"errors"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"sync"
)

type Call struct {
	ServiceMethod string
	Seq           uint64
	Error         error
	Args, Reply   interface{}
	Done          chan *Call
}

func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	io.Closer
	cc       codec.ICodec
	seq      uint64
	sending  *sync.Mutex
	mu       *sync.Mutex
	pending  map[uint64]*Call // pending calls
	shutdown bool             // server side error
}

var _ io.Closer = (*Client)(nil)

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
	client := &Client{
		cc:      cc,
		sending: new(sync.Mutex),
		mu:      new(sync.Mutex),
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	return client.cc.Close()
}

func (client *Client) Call(ServiceMethod string, args, reply interface{}) error {
	call := <-client.Go(ServiceMethod, args, reply, make(chan *Call, 1)).Done
	return call.Error
}

func (client *Client) Go(ServiceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	}
	call := &Call{
		ServiceMethod: ServiceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	go client.send(call)
	return call
}

func (client *Client) send(call *Call) {
	client.sending.Lock()
	defer client.sending.Unlock()

	h := &codec.Header{
		ServiceMethod: call.ServiceMethod,
		Seq:           client.seq,
		Error:         "",
	}
	call.Seq = client.seq
	
	client.seq++
	if err := client.cc.Write(h, call.Args); err != nil {
		call.done()
		log.Println("rpc client: send call failed", err)
		return
	}

	client.pending[call.Seq] = call
}

func (client *Client) popCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

// iterate pending map and stop all calls
func (client *Client) terminateAll(err error) {
	client.sending.Lock() // send() would change the map's size
	defer client.mu.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()

	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
	client.shutdown = true
}

// keep receiving from server
func (client *Client) receive() {
	var err error
	for err == nil {
		var h codec.Header
		if client.cc.ReadHeader(&h); err != nil {
			log.Println("rpc client: read header error", err)
			break
		}
		call := client.popCall(h.Seq)
		switch {
		case call == nil:
			err = client.cc.ReadBody(nil)
		case h.Error != "":
			err = client.cc.ReadBody(nil)
			call.Error = errors.New(h.Error)
			call.done()
		default:
			if err = client.cc.ReadBody(call.Reply); err != nil {
				call.Error = err
			}
			call.done()
		}
	}
	client.terminateAll(err)
}
