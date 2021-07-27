package tokenizer

import (
	"fmt"

	"github.com/pqkallio/nand2tetris-jack-compiler/vm"
)

type TokenType int

const (
	Keyword TokenType = iota
	Symbol
	IntegerConstant
	StringConstant
	Identifier
	Comment
	Error
	EOF
)

func (t TokenType) String() string {
	switch t {
	case Keyword:
		return "keyword"
	case Symbol:
		return "symbol"
	case IntegerConstant:
		return "integerConstant"
	case StringConstant:
		return "stringConstant"
	case Identifier:
		return "identifier"
	case Comment:
		return "comment"
	case Error:
		return "error"
	case EOF:
		return "EOF"
	default:
		return "unknown"
	}
}

type Terminal struct {
	Type            TokenType `xml:"-"`
	Keyword         string    `xml:"keyword,omitempty"`
	Symbol          string    `xml:"symbol,omitempty"`
	IntegerConstant string    `xml:"integerConstant,omitempty"`
	StringConstant  string    `xml:"stringConstant,omitempty"`
	Identifier      string    `xml:"identifier,omitempty"`
}

func (t Terminal) IsOfType(tt TokenType) bool {
	return t.Type == tt
}

func (t Terminal) IsAnyOf(tts ...TokenType) bool {
	for _, tt := range tts {
		if t.Type == tt {
			return true
		}
	}

	return false
}

func (t Terminal) IsKeyword(ks ...string) bool {
	if t.Type != Keyword {
		return false
	}

	for _, k := range ks {
		if t.Keyword == k {
			return true
		}
	}

	return false
}

func (t Terminal) IsSymbol(ss ...string) bool {
	if t.Type != Symbol {
		return false
	}

	for _, s := range ss {
		if t.Symbol == s {
			return true
		}
	}

	return false
}

func (t Terminal) VMOp() vm.Op {
	switch t.Symbol {
	case "+":
		return vm.Add
	case "-":
		return vm.Sub
	case "&":
		return vm.And
	case "|":
		return vm.Or
	case "=":
		return vm.Eq
	case "<":
		return vm.Lt
	case ">":
		return vm.Gt
	default:
		return ""
	}
}

func (t Terminal) String() string {
	s := fmt.Sprintf("{Type:%s", t.Type)

	if len(t.Keyword) != 0 {
		s = fmt.Sprintf("%s Keyword:%s", s, t.Keyword)
	}

	if len(t.Symbol) != 0 {
		s = fmt.Sprintf("%s Symbol:%s", s, t.Symbol)
	}

	if len(t.IntegerConstant) != 0 {
		s = fmt.Sprintf("%s Integer:%s", s, t.IntegerConstant)
	}

	if len(t.StringConstant) != 0 {
		s = fmt.Sprintf("%s String:%s", s, t.StringConstant)
	}

	if len(t.Identifier) != 0 {
		s = fmt.Sprintf("%s Identifier:%s", s, t.Identifier)
	}

	return fmt.Sprintf("%s}", s)
}
