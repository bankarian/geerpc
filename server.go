package geerpc

import (
	"encoding/json"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

type Option struct {
	MagicNumber int
	CodecType   codec.CodecType
}

const MagicNumber = 0x83d38e

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

type request struct {
	h *codec.Header
	// we would use argv and replyv as value receivers,
	// so use reflect.Value instead of interface{}
	argv, replyv reflect.Value
}

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

func (s *Server) Accept(lis net.Listener) {
	// reactor
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error", err)
			return
		}
		go s.ServeConn(conn)
	}
}

func (s *Server) ServeConn(conn io.ReadWriteCloser) {
	// 1. json decode Option, getting codec function
	// 2. read request: decode header, then body(argv)
	// 3. handle request: read argv, write replyv, send response
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: decode option error", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid request magic number %x", opt.MagicNumber)
		return
	}

	cc := codec.NewCodecFuncMap[opt.CodecType](conn)
	s.serve(cc)
}

func (s *Server) serve(cc codec.ICodec) {
	// read request in order
	// handle request concurrently
	wg := new(sync.WaitGroup)
	sending := new(sync.Mutex)
	for {
		var req request
		if err := s.readRequest(cc, &req); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			req.h.Error = err.Error()
			s.sendResponse(cc, &req, sending)
			continue
		}
		wg.Add(1)
		go s.handleRequest(cc, &req, wg, sending)
	}
	wg.Wait()
}

func (s *Server) readRequest(cc codec.ICodec, req *request) error {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error", err)
		}
		return err
	}
	req.h = &h

	// TODO: assume that the type is string currently
	req.argv = reflect.New(reflect.TypeOf(""))
	if err := cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv error", err)
		return err
	}

	return nil
}

func (s *Server) handleRequest(cc codec.ICodec, req *request, wg *sync.WaitGroup, sending *sync.Mutex) {
	defer wg.Done()
	// TODO: just send back a string currently
	req.replyv = reflect.ValueOf(req.argv.Interface())
	s.sendResponse(cc, req, sending)
}

func (s *Server) sendResponse(cc codec.ICodec, req *request, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(req.h, req.replyv.Interface()); err != nil {
		log.Println("rpc server: send response error", err)
	}
}
