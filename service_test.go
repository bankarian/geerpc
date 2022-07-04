package geerpc

import (
	"testing"
)

type Foo int

func (foo *Foo) Hello(arg string, reply *string) error {
	return nil
}

func (foo *Foo) wow() {
}

func TestRegister(t *testing.T) {
	var foo Foo
	s := newService(&foo)
	if len(s.methods) != 1 {
		t.Error("register error")
	}
	if s.methods["Hello"] == nil {
		t.Error("register failed")
	}
}
