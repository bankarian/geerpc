package geerpc

import (
	"testing"
)

type Foo int

type Arg struct {
	A, B int
}

type Reply struct {
	Sum int
}

func (foo *Foo) Hello(arg string, reply *string) error {
	return nil
}

func (foo *Foo) Sum(arg Arg, reply *Reply) error {
	reply.Sum = arg.A + arg.B
	return nil
}

func (foo *Foo) wow() {
}

func TestRegister(t *testing.T) {
	var foo Foo
	s := newService(&foo)
	if len(s.methods) != 2 {
		t.Error("register error")
	}
	if s.methods["Hello"] == nil || s.methods["Sum"] == nil {
		t.Error("register failed")
	}
}

func TestCallMethod(t *testing.T) {
	var foo Foo
	s := newService(&foo)
	var reply Reply
	s.call("Sum", Arg{2, 3}, &reply)
	if reply.Sum != 5 {
		t.Error("call Sum(2, 3) failed")
	}
}
