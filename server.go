package geerpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"geerpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"strings"
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
	ser          *service
	mName        string
}

type Server struct {
	services map[string]*service
}

func NewServer() *Server {
	return &Server{
		services: make(map[string]*service),
	}
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
	req.h = &h
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error", err)
		}
		return err
	}

	ser, method, err := s.findServiceMethod(&h)
	if err != nil {
		log.Println("rpc server error:", err)
		return err
	}
	req.argv = method.newArgRcvr()
	req.replyv = method.newReplyRcvr()
	req.ser = ser
	req.mName = method.method.Name

	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		// make sure the arg receiver is a pointer
		argvi = req.argv.Addr().Interface()
	}
	if err := cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read argv error", err)
		return err
	}
	return nil
}

func (s *Server) handleRequest(cc codec.ICodec, req *request, wg *sync.WaitGroup, sending *sync.Mutex) {
	defer wg.Done()
	req.ser.call(req.mName, req.argv.Interface(), req.replyv.Interface())
	s.sendResponse(cc, req, sending)
}

func (s *Server) sendResponse(cc codec.ICodec, req *request, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	var body interface{}
	if req.replyv == reflect.ValueOf(nil) {
		body = struct{}{}
	} else {
		body = req.replyv.Interface()
	}
	if err := cc.Write(req.h, body); err != nil {
		log.Println("rpc server: send response error", err)
	}
}

func (s *Server) findServiceMethod(h *codec.Header) (*service, *methodType, error) {
	ss := strings.Split(h.ServiceMethod, ".")
	if len(ss) != 2 {
		return nil, nil, errors.New("Service.Method ill-formatted")
	}
	serviceName, methodName := ss[0], ss[1]
	service, ok := s.services[serviceName]
	if !ok {
		return nil, nil, fmt.Errorf("service <%s> not found", serviceName)
	}
	method, ok := service.methods[methodName]
	if !ok {
		return nil, nil, fmt.Errorf("method <%s> not found", methodName)
	}
	return service, method, nil
}

// Register registers a service to the default server
func Register(rcvr interface{}) {
	ser := newService(rcvr)
	DefaultServer.services[ser.name] = ser
}
