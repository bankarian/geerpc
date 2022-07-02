package geerpc

import (
	"encoding/json"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
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
	// we would use argv and replyv in details,
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

func (s *Server) ServeConn(conn io.ReadWriteCloser) error {
	// 1. json decode Option, getting codec function
	// 2. read request: decode header, then body(argv)
	// 3. handle request: read argv, write replyv, send response
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: decode option error", err)
		return err
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid request magic number %x", opt.MagicNumber)
		return fmt.Errorf("invalid request magic number %x", opt.MagicNumber)
	}
	cc := codec.NewCodecFuncMap[opt.CodecType](conn)
	for {
		var req request
		if err := s.readRequest(cc, &req); err != nil {
			if err == io.ErrUnexpectedEOF || err == io.EOF {
				break
			}
			return err
		}
		if err := s.handleRequest(cc, &req); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) readRequest(cc codec.ICodec, req *request) error {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		log.Println("rpc server: read header error", err)
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

func (s *Server) handleRequest(cc codec.ICodec, req *request) error {
	// TODO: just send back a string currently
	req.replyv = reflect.ValueOf(req.argv.Interface())

	// send response
	if err := cc.Write(req.h, req.replyv.Interface()); err != nil {
		log.Println("rpc server: send response error", err)
		return err
	}
	return nil
}
