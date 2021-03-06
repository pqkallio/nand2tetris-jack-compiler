package symbols

import "github.com/pqkallio/nand2tetris-jack-compiler/vm"

type Scope int

const (
	Field Scope = iota
	Static
	Argument
	Local
)

func (s Scope) String() string {
	switch s {
	case Field:
		return "THIS"
	case Static:
		return "STATIC"
	case Argument:
		return "ARG"
	case Local:
		return "LOCAL"
	default:
		return "UNKNOWN"
	}
}

func (s Scope) In(ss ...Scope) bool {
	for _, s2 := range ss {
		if s2 == s {
			return true
		}
	}

	return false
}

func (s Scope) ToVMMemSeg() vm.MemSeg {
	switch s {
	case Field:
		return vm.This
	case Static:
		return vm.Static
	case Argument:
		return vm.Arg
	default:
		return vm.Local
	}
}

type Entry struct {
	Name  string
	Scope Scope
	Type  string
	Idx   uint
}
