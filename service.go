package geerpc

import (
	"go/ast"
	"log"
	"reflect"
)

type methodType struct {
	argType   reflect.Type
	replyType reflect.Type
	method    reflect.Method
}

func (m *methodType) newArgRcvr() reflect.Value {
	var arg reflect.Value
	// arg can be pointer, or value
	if m.argType.Kind() == reflect.Ptr {
		arg = reflect.New(m.argType.Elem())
	} else {
		arg = reflect.New(m.argType).Elem()
	}
	return arg
}

func (m *methodType) newReplyRcvr() reflect.Value {
	// reply must be a pointer
	reply := reflect.New(m.replyType.Elem())
	switch m.replyType.Elem().Kind() {
	case reflect.Map:
		reply.Elem().Set(reflect.MakeMap(m.replyType.Elem()))
	case reflect.Slice:
		reply.Elem().Set(reflect.MakeSlice(m.replyType.Elem(), 0, 0))
	}
	return reply
}

type service struct {
	name    string
	typ     reflect.Type
	rcvr     reflect.Value
	methods map[string]*methodType
}

func newService(rcvr interface{}) *service {
	s := new(service)
	s.rcvr = reflect.ValueOf(rcvr)
	s.typ = reflect.TypeOf(rcvr)
	s.name = reflect.Indirect(s.rcvr).Type().Name()
	if !ast.IsExported(s.name) {
		log.Fatalf("rpc server: invalid service name %s", s.name)
	}
	s.registerMethods()
	return s
}

func (s *service) registerMethods() {
	// get all Exported methods
	// check restriction, ignore illegal methods
	// 1. three params: self arg reply
	// 2. arg & reply exported or builtin
	// 3. reply is pointer
	// 4. one return value: error
	s.methods = make(map[string]*methodType)
	for i := 0; i < s.typ.NumMethod(); i++ {
		m := s.typ.Method(i)
		mType := m.Type
		if mType.NumIn() != 3 {
			continue
		}
		argType, replyType := mType.In(1), mType.In(2)
		if !isExportedOrBuiltin(argType) || !isExportedOrBuiltin(replyType) {
			continue
		}
		if !isPtr(replyType) {
			continue
		}
		if mType.NumOut() != 1 || mType.Out(0) != reflect.TypeOf((*error)(nil)).Elem() {
			continue
		}
		s.methods[m.Name] = &methodType{
			method: m,
			argType: argType,
			replyType: replyType,
		}
		log.Printf("rpc server: register method %s.%s\n", s.name, m.Name)
	}
}

func (s *service) call(name string, arg, reply interface{}) error {
	method := s.methods[name]
	res := method.method.Func.Call([]reflect.Value{
		s.rcvr, reflect.ValueOf(arg),reflect.ValueOf(reply)})
	if errVal := res[0].Interface(); errVal != nil {
		return errVal.(error)
	}
	return nil
}

func isExportedOrBuiltin(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

func isPtr(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr
}
