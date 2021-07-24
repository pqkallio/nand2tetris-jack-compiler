package compilationengine

import (
	"fmt"
	"os"

	"github.com/pqkallio/nand2tetris-jack-compiler/symbols"
	"github.com/pqkallio/nand2tetris-jack-compiler/tokenizer"
)

type Service struct {
	t  *tokenizer.Service
	st *symbols.Table
	f  *os.File
	d  string
}

func New(t *tokenizer.Service, output *os.File) *Service {
	return &Service{t, symbols.New(), output, ""}
}

func (s *Service) Compile() error {
	t, err := s.eatKeyword("class")
	if err != nil {
		return err
	}

	return s.compileClass(t)
}

func (s *Service) compileClass(t tokenizer.Terminal) error {
	tag := "class"

	s.write(s.openingTag(tag))
	s.indent()

	s.write(s.keyword(t))

	t, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	s.write(s.identifier(t))

	t, err = s.eatSymbol("{")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	for {
		err = s.compileClassVarDec()
		if err != nil {
			break
		}
	}

	for {
		err = s.compileSubroutineDec()
		if err != nil {
			break
		}
	}

	t, err = s.eatSymbol("}")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileSubroutineDec() error {
	tag := "subroutineDec"

	t, err := s.eatKeyword("constructor", "function", "method")
	if err != nil {
		return err
	}

	s.write(s.openingTag(tag))
	s.indent()

	s.write(s.keyword(t))

	t, err = s.eatReturnType()
	if err != nil {
		return err
	}

	s.write(s.tType(t))

	t, err = s.eatIdentifier()
	if err != nil {
		return err
	}

	s.write(s.identifier(t))

	s.st.SwitchSubroutineTo(s.identifier(t))

	t, err = s.eatSymbol("(")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	err = s.compileParameterList()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol(")")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	err = s.compileSubroutineBody()
	if err != nil {
		return err
	}

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileSubroutineBody() error {
	tag := "subroutineBody"

	s.write(s.openingTag(tag))
	s.indent()

	t, err := s.eatSymbol("{")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	for {
		err = s.compileVarDec()
		if err != nil {
			break
		}
	}

	err = s.compileStatements()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol("}")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileStatements() error {
	tag := "statements"

	s.write(s.openingTag(tag))
	s.indent()

	for {
		t, err := s.eatKeyword("let", "if", "while", "do", "return")
		if err != nil {
			break
		}

		switch k := t.Keyword; k {
		case "let":
			err = s.compileLetStatement(t)
		case "if":
			err = s.compileIfStatement(t)
		case "while":
			err = s.compileWhileStatement(t)
		case "do":
			err = s.compileDoStatement(t)
		case "return":
			err = s.compileReturnStatement(t)
		}

		if err != nil {
			return err
		}
	}

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileExpressionList() error {
	tag := "expressionList"

	s.write(s.openingTag(tag))
	s.indent()

	_, err := s.eatSymbol(")")
	if err != nil {
		for {
			err = s.compileExpression()
			if err != nil {
				return err
			}

			t, err := s.eatSymbol(",")
			if err != nil {
				break
			}

			s.write(s.symbol(t))
		}
	} else {
		s.t.Rewind(0)
	}

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileSubroutineCall() error {
	t, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	s.write(s.identifier(t))

	t, err = s.eatSymbol(".", "(")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	switch sym := t.Symbol; sym {
	case ".":
		t, err = s.eatIdentifier()
		if err != nil {
			return err
		}

		s.write(s.identifier(t))

		t, err = s.eatSymbol("(")
		if err != nil {
			return err
		}

		s.write(s.symbol(t))

		fallthrough
	case "(":
		err = s.compileExpressionList()
		if err != nil {
			return err
		}

		t, err = s.eatSymbol(")")
		if err != nil {
			return err
		}

		s.write(s.symbol(t))
	}

	return nil
}

func (s *Service) compileTerm() error {
	tag := "term"

	s.write(s.openingTag(tag))
	s.indent()

	t := s.eat()

	switch tt := t.Type; tt {
	case tokenizer.IntegerConstant:
		s.write(s.integerConstant(t))
	case tokenizer.StringConstant:
		s.write(s.stringConstant(t))
	case tokenizer.Keyword:
		s.write(s.keyword(t))
	case tokenizer.Identifier:
		t2, err := s.eatSymbol("(", "[", ".")
		if err != nil {
			s.write(s.identifier(t))
			e := s.st.Get(t.Identifier)

			s.write(s.memEntry(e))

			break
		}

		switch sym := t2.Symbol; sym {
		case ".":
			fallthrough
		case "(":
			s.t.Rewind(1)

			err = s.compileSubroutineCall()
			if err != nil {
				return err
			}
		case "[":
			s.write(s.identifier(t))

			e := s.st.Get(t.Identifier)

			s.write(s.memEntry(e))

			s.write(s.symbol(t2))

			err = s.compileExpression()
			if err != nil {
				return err
			}

			t, err = s.eatSymbol("]")
			if err != nil {
				return err
			}

			s.write(s.symbol(t))
		}
	case tokenizer.Symbol:
		s.write(s.symbol(t))

		switch sym := t.Symbol; sym {
		case "(":
			err := s.compileExpression()
			if err != nil {
				return err
			}

			t, err = s.eatSymbol(")")
			if err != nil {
				return err
			}

			s.write(s.symbol(t))
		case "-":
			fallthrough
		case "~":
			err := s.compileTerm()
			if err != nil {
				return err
			}
		}
	}

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileExpression() error {
	tag := "expression"

	s.write(s.openingTag(tag))
	s.indent()

	err := s.compileTerm()
	if err != nil {
		return err
	}

	t, err := s.eatBinOp()
	if err == nil {
		s.write(s.symbol(t))

		err = s.compileTerm()
		if err != nil {
			return err
		}
	}

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileReturnStatement(t tokenizer.Terminal) error {
	tag := "returnStatement"

	s.write(s.openingTag(tag))
	s.indent()

	s.write(s.keyword(t))

	_, err := s.eatSymbol(";")
	if err != nil {
		s.compileExpression()
	} else {
		s.t.Rewind(0)
	}

	t, err = s.eatSymbol(";")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileDoStatement(t tokenizer.Terminal) error {
	tag := "doStatement"

	s.write(s.openingTag(tag))
	s.indent()

	s.write(s.keyword(t))

	err := s.compileSubroutineCall()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol(";")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileWhileStatement(t tokenizer.Terminal) error {
	tag := "whileStatement"

	s.write(s.openingTag(tag))
	s.indent()

	s.write(s.keyword(t))

	t, err := s.eatSymbol("(")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	err = s.compileExpression()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol(")")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	t, err = s.eatSymbol("{")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	err = s.compileStatements()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol("}")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	s.deindent()
	s.write(s.closingTag(tag))

	return nil

}

func (s *Service) compileIfStatement(t tokenizer.Terminal) error {
	tag := "ifStatement"

	s.write(s.openingTag(tag))
	s.indent()

	s.write(s.keyword(t))

	t, err := s.eatSymbol("(")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	err = s.compileExpression()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol(")")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	t, err = s.eatSymbol("{")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	err = s.compileStatements()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol("}")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	t, err = s.eatKeyword("else")
	if err == nil {
		s.write(s.keyword(t))

		t, err = s.eatSymbol("{")
		if err != nil {
			return err
		}

		s.write(s.symbol(t))

		err = s.compileStatements()
		if err != nil {
			return err
		}

		t, err = s.eatSymbol("}")
		if err != nil {
			return err
		}

		s.write(s.symbol(t))
	}

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileLetStatement(t tokenizer.Terminal) error {
	tag := "letStatement"

	s.write(s.openingTag(tag))
	s.indent()

	s.write(s.keyword(t))

	t, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	s.write(s.identifier(t))

	e := s.st.Get(t.Identifier)
	s.write(s.memEntry(e))

	t, err = s.eatSymbol("[", "=")
	if err != nil {
		return nil
	}

	s.write(s.symbol(t))

	if t.Symbol == "[" {
		err = s.compileExpression()
		if err != nil {
			return err
		}

		t, err = s.eatSymbol("]")
		if err != nil {
			return err
		}

		s.write(s.symbol(t))

		t, err = s.eatSymbol("=")
		if err != nil {
			return err
		}

		s.write(s.symbol(t))
	}

	err = s.compileExpression()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol(";")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileVarDec() error {
	tag := "varDec"

	t, err := s.eatKeyword("var")
	if err != nil {
		return err
	}

	s.write(s.openingTag(tag))
	s.indent()

	s.write(s.keyword(t))

	tp, err := s.eatVarType()
	if err != nil {
		return err
	}

	s.write(s.tType(tp))

	id, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	s.write(s.identifier(id))

	e := s.st.Define(id.Identifier, s.getType(tp), "local")

	s.write(s.memEntry(e))

	t, err = s.eatSymbol(",", ";")
	if err != nil {
		return err
	}

	for t.Symbol == "," {
		s.write(s.symbol(t))

		id, err = s.eatIdentifier()
		if err != nil {
			return err
		}

		s.write(s.identifier(id))

		e = s.st.Define(id.Identifier, s.getType(tp), "local")

		s.write(s.memEntry(e))

		t, err = s.eatSymbol(",", ";")
		if err != nil {
			return err
		}
	}

	s.write(s.symbol(t))

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileParameterList() error {
	tag := "parameterList"

	s.write(s.openingTag(tag))
	s.indent()

	tp, err := s.eatVarType()
	if err != nil {
		s.deindent()
		s.write(s.closingTag(tag))

		return nil
	}

	s.write(s.tType(tp))

	id, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	s.write(s.identifier(id))

	e := s.st.Define(id.Identifier, s.getType(tp), "arg")
	s.write(s.memEntry(e))

	for {
		t, err := s.eatSymbol(",")
		if err != nil {
			break
		}

		s.write(s.symbol(t))

		tp, err = s.eatVarType()
		if err != nil {
			return err
		}

		s.write(s.tType(tp))

		id, err = s.eatIdentifier()
		if err != nil {
			return err
		}

		s.write(s.identifier(id))

		e := s.st.Define(id.Identifier, s.getType(tp), "arg")
		s.write(s.memEntry(e))
	}

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileClassVarDec() error {
	tag := "classVarDec"

	sc, err := s.eatKeyword("static", "field")
	if err != nil {
		return err
	}

	s.write(s.openingTag(tag))
	s.indent()

	s.write(s.keyword(sc))

	tp, err := s.eatVarType()
	if err != nil {
		return err
	}

	s.write(s.tType(tp))

	id, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	s.write(s.identifier(id))

	e := s.st.Define(id.Identifier, s.getType(tp), sc.Keyword)
	s.write(s.memEntry(e))

	t, err := s.eatSymbol(",", ";")
	if err != nil {
		return err
	}

	for t.Symbol != ";" {
		s.write(s.symbol(t))

		id, err = s.eatIdentifier()
		if err != nil {
			return err
		}

		s.write(s.identifier(id))

		e = s.st.Define(id.Identifier, s.getType(tp), sc.Keyword)
		s.write(s.memEntry(e))

		t, err = s.eatSymbol(",", ";")
		if err != nil {
			return err
		}
	}

	s.write(s.symbol(t))

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) write(str string) {
	s.f.WriteString(str)
}

func (s *Service) indent() {
	s.d = s.d + "  "
}

func (s *Service) deindent() {
	if len(s.d) > 1 {
		s.d = s.d[:len(s.d)-2]
	}
}

func (s *Service) eat() tokenizer.Terminal {
	s.t.Advance()
	return s.t.ConsumeToken()
}

func (s *Service) eatBinOp() (tokenizer.Terminal, error) {
	return s.eatSymbol("+", "-", "*", "/", "&", "|", "<", ">", "=")
}

func (s *Service) eatSymbol(ss ...string) (tokenizer.Terminal, error) {
	s.t.Advance()

	if s.t.Token().IsSymbol(ss...) {
		return s.t.ConsumeToken(), nil
	}

	return tokenizer.Terminal{}, fmt.Errorf("expected one of symbols %v but token was %s", ss, s.t.Token())
}

func (s *Service) eatKeyword(ks ...string) (tokenizer.Terminal, error) {
	s.t.Advance()

	if s.t.Token().IsKeyword(ks...) {
		return s.t.ConsumeToken(), nil
	}

	return tokenizer.Terminal{}, fmt.Errorf("expected one of keywords %v but token was %s", ks, s.t.Token())
}

func (s *Service) eatIdentifier() (tokenizer.Terminal, error) {
	s.t.Advance()

	if s.t.Token().IsOfType(tokenizer.Identifier) {
		return s.t.ConsumeToken(), nil
	}

	return tokenizer.Terminal{}, fmt.Errorf("expected identifier but token was %s", s.t.Token())
}

func (s *Service) eatType(ts ...string) (tokenizer.Terminal, error) {
	s.t.Advance()

	if s.t.Token().IsOfType(tokenizer.Identifier) {
		return s.t.ConsumeToken(), nil
	}

	if s.t.Token().IsKeyword(ts...) {
		return s.t.ConsumeToken(), nil
	}

	return tokenizer.Terminal{}, fmt.Errorf("expected a type but token was %s", s.t.Token())
}

func (s *Service) eatVarType() (tokenizer.Terminal, error) {
	return s.eatType("char", "boolean", "int")
}

func (s *Service) eatReturnType() (tokenizer.Terminal, error) {
	return s.eatType("char", "boolean", "int", "void")
}

func (s *Service) symbol(t tokenizer.Terminal) string {
	return s.tagged("symbol", getEscapedSymbol(t.Symbol))
}

func (s *Service) keyword(t tokenizer.Terminal) string {
	return s.tagged("keyword", t.Keyword)
}

func (s *Service) identifier(t tokenizer.Terminal) string {
	return s.tagged("identifier", t.Identifier)
}

func (s *Service) integerConstant(t tokenizer.Terminal) string {
	return s.tagged("integerConstant", t.IntegerConstant)
}

func (s *Service) stringConstant(t tokenizer.Terminal) string {
	return s.tagged("stringConstant", t.StringConstant)
}

func (s *Service) memEntry(e *symbols.Entry) string {
	return s.tagged("memSeg", e.Scope.String()) + s.tagged("memIdx", fmt.Sprintf("%d", e.Idx))
}

func (s *Service) tType(t tokenizer.Terminal) string {
	if t.Type == tokenizer.Keyword {
		return s.keyword(t)
	}

	return s.identifier(t)
}

func (s *Service) getType(t tokenizer.Terminal) string {
	if t.Type == tokenizer.Keyword {
		return t.Keyword
	}

	return t.Identifier
}

func (s *Service) tagged(tag, val string) string {
	return fmt.Sprintf("%s<%s> %s </%s>\n", s.d, tag, val, tag)
}

func (s *Service) openingTag(tag string) string {
	return s.d + "<" + tag + ">\n"
}

func (s *Service) closingTag(tag string) string {
	return s.d + "</" + tag + ">\n"
}
