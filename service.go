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

type service struct {
	name    string
	typ     reflect.Type
	val     reflect.Value
	methods map[string]*methodType
}

func newService(rcvr interface{}) *service {
	s := new(service)
	s.val = reflect.ValueOf(rcvr)
	s.typ = reflect.TypeOf(rcvr)
	s.name = reflect.Indirect(s.val).Type().Name()
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

func isExportedOrBuiltin(t reflect.Type) bool {
	return ast.IsExported(t.Name()) || t.PkgPath() == ""
}

func isPtr(t reflect.Type) bool {
	return t.Kind() == reflect.Ptr
}
