package compilationengine

import (
	"fmt"
	"os"
	"strconv"

	"github.com/pqkallio/nand2tetris-jack-compiler/symbols"
	"github.com/pqkallio/nand2tetris-jack-compiler/tokenizer"
	"github.com/pqkallio/nand2tetris-jack-compiler/vm"
)

type Service struct {
	tokenizer   *tokenizer.Service
	symbolTable *symbols.Table
	vmWriter    *vm.Writer
	xmlFile     *os.File
	indentation string
	className   string
}

func New(t *tokenizer.Service, vmWriter *vm.Writer, xmlOut *os.File) *Service {
	return &Service{t, symbols.New(), vmWriter, xmlOut, "", ""}
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
	s.className = t.Identifier

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

	tt, err := s.eatKeyword("constructor", "function", "method")
	if err != nil {
		return err
	}

	s.write(s.openingTag(tag))
	s.indent()

	s.write(s.keyword(tt))

	t, err := s.eatReturnType()
	if err != nil {
		return err
	}

	s.write(s.tType(t))

	t, err = s.eatIdentifier()
	if err != nil {
		return err
	}

	s.write(s.identifier(t))
	funcName := t.Identifier

	s.symbolTable.SwitchSubroutineTo(s.identifier(t))

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

	err = s.compileSubroutineBody(funcName, tt.Keyword)
	if err != nil {
		return err
	}

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileSubroutineBody(funcName, funcType string) error {
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

	nLocals := s.symbolTable.GetSymbolCount(symbols.Local)
	s.vmWriter.WriteFunc(s.className+"."+funcName, nLocals)

	switch funcType {
	case "method":
		// set the correct object to "this"
		s.vmWriter.WritePush(vm.Arg, 0)
		s.vmWriter.WritePop(vm.Pointer, 0)
	case "constructor":
		// allocate memory for the object
		nFields := s.symbolTable.GetSymbolCount(symbols.Field)
		s.vmWriter.WritePush(vm.Const, nFields)
		s.vmWriter.WriteCall("Memory.alloc", 1)
		s.vmWriter.WritePop(vm.Pointer, 0)
	}

	err = s.compileStatements()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol("}")
	if err != nil {
		return err
	}

	if funcType == "constructor" {
		s.vmWriter.WritePush(vm.Pointer, 0)
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

func (s *Service) compileExpressionList() (uint, error) {
	tag := "expressionList"
	nArgs := uint(0)

	s.write(s.openingTag(tag))
	s.indent()

	_, err := s.eatSymbol(")")
	if err != nil {
		for {
			err = s.compileExpression()
			if err != nil {
				return 0, err
			}

			nArgs += 1

			t, err := s.eatSymbol(",")
			if err != nil {
				break
			}

			s.write(s.symbol(t))
		}
	} else {
		s.tokenizer.Rewind(0)
	}

	s.deindent()
	s.write(s.closingTag(tag))

	return nArgs, nil
}

func (s *Service) compileSubroutineCall() error {
	t, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	id := t.Identifier

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

		id += "." + t.Identifier

		s.write(s.identifier(t))

		t, err = s.eatSymbol("(")
		if err != nil {
			return err
		}

		s.write(s.symbol(t))

		fallthrough
	case "(":
		nArgs, err := s.compileExpressionList()
		if err != nil {
			return err
		}

		t, err = s.eatSymbol(")")
		if err != nil {
			return err
		}

		s.vmWriter.WriteCall(id, nArgs)
		s.write(s.symbol(t))
	}

	return nil
}

func (s *Service) pushStringConstant(str string) {
	strLen := uint(len(str))

	s.vmWriter.WritePush(vm.Const, strLen)
	s.vmWriter.WriteCall("String.new", 1)

	for _, c := range str {
		s.vmWriter.WritePush(vm.Const, uint(c))
		s.vmWriter.WriteCall("String.appendChar", 2)
	}
}

func (s *Service) pushKeywordConstant(c string) {
	switch c {
	case "true":
		s.vmWriter.WritePush(vm.Const, 1)
		s.vmWriter.WriteArithmetic(vm.Neg)
	default:
		s.vmWriter.WritePush(vm.Const, 0)
	}
}

func (s *Service) compileTerm() error {
	tag := "term"

	s.write(s.openingTag(tag))
	s.indent()

	t := s.eat()

	switch tt := t.Type; tt {
	case tokenizer.IntegerConstant:
		s.write(s.integerConstant(t))
		i, _ := strconv.Atoi(t.IntegerConstant)
		s.vmWriter.WritePush(vm.Const, uint(i))
	case tokenizer.StringConstant:
		s.write(s.stringConstant(t))
		s.pushStringConstant(t.StringConstant)
	case tokenizer.Keyword:
		s.write(s.keyword(t))
		s.pushKeywordConstant(t.Keyword)
	case tokenizer.Identifier:
		t2, err := s.eatSymbol("(", "[", ".")
		if err != nil {
			s.write(s.identifier(t))
			e := s.symbolTable.Get(t.Identifier)

			s.write(s.memEntry(e))
			s.vmWriter.WritePush(e.Scope.ToVMMemSeg(), e.Idx)

			break
		}

		switch sym := t2.Symbol; sym {
		case ".":
			fallthrough
		case "(":
			s.tokenizer.Rewind(1)

			err = s.compileSubroutineCall()
			if err != nil {
				return err
			}
		case "[":
			s.write(s.identifier(t))

			e := s.symbolTable.Get(t.Identifier)

			s.write(s.memEntry(e))

			s.vmWriter.WritePush(e.Scope.ToVMMemSeg(), e.Idx)

			s.write(s.symbol(t2))

			err = s.compileExpression()
			if err != nil {
				return err
			}

			t, err = s.eatSymbol("]")
			if err != nil {
				return err
			}

			s.vmWriter.WriteArithmetic(vm.Add)
			s.vmWriter.WritePop(vm.Pointer, 1)
			s.vmWriter.WritePush(vm.That, 0)

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

		switch t.Symbol {
		case "*":
			s.vmWriter.WriteCall("Math.multiply", 2)
		case "/":
			s.vmWriter.WriteCall("Math.divide", 2)
		default:
			s.vmWriter.WriteArithmetic(t.VMOp())
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
		s.tokenizer.Rewind(0)
	}

	t, err = s.eatSymbol(";")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	s.vmWriter.WriteReturn()

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

	s.vmWriter.WritePop(vm.Temp, 0)

	s.write(s.symbol(t))

	s.deindent()
	s.write(s.closingTag(tag))

	return nil
}

func (s *Service) compileWhileStatement(t tokenizer.Terminal) error {
	tag := "whileStatement"

	lblFalse := s.vmWriter.RegisterLabel("IF_FALSE")
	lblTrue := s.vmWriter.RegisterLabel("IF_TRUE")

	s.write(s.openingTag(tag))
	s.indent()

	s.write(s.keyword(t))

	s.vmWriter.WriteLabel(lblTrue)

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

	s.vmWriter.WriteArithmetic(vm.Not)
	s.vmWriter.WriteIf(lblFalse)

	t, err = s.eatSymbol("{")
	if err != nil {
		return err
	}

	s.write(s.symbol(t))

	err = s.compileStatements()
	if err != nil {
		return err
	}

	s.vmWriter.WriteGoto(lblTrue)

	t, err = s.eatSymbol("}")
	if err != nil {
		return err
	}

	s.vmWriter.WriteLabel(lblFalse)

	s.write(s.symbol(t))

	s.deindent()
	s.write(s.closingTag(tag))

	return nil

}

func (s *Service) compileIfStatement(t tokenizer.Terminal) error {
	tag := "ifStatement"

	lblFalse := s.vmWriter.RegisterLabel("IF_FALSE")
	lblTrue := s.vmWriter.RegisterLabel("IF_TRUE")

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

	s.vmWriter.WriteArithmetic(vm.Not)
	s.vmWriter.WriteIf(lblFalse)

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

	s.vmWriter.WriteGoto(lblTrue)

	s.write(s.symbol(t))

	s.vmWriter.WriteLabel(lblFalse)

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

	s.vmWriter.WriteLabel(lblTrue)

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

	e := s.symbolTable.Get(t.Identifier)
	target := vm.MemEntry{e.Scope.ToVMMemSeg(), e.Idx}
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

		s.vmWriter.WriteArithmetic(vm.Add)
		s.vmWriter.WritePop(vm.Pointer, 1)

		target = vm.MemEntry{vm.That, 0}
	}

	err = s.compileExpression()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol(";")
	if err != nil {
		return err
	}

	s.vmWriter.WritePop(target.Seg, target.Idx)

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

	e := s.symbolTable.Define(id.Identifier, s.getType(tp), "local")

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

		e = s.symbolTable.Define(id.Identifier, s.getType(tp), "local")

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

	e := s.symbolTable.Define(id.Identifier, s.getType(tp), "arg")
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

		e := s.symbolTable.Define(id.Identifier, s.getType(tp), "arg")
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

	e := s.symbolTable.Define(id.Identifier, s.getType(tp), sc.Keyword)
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

		e = s.symbolTable.Define(id.Identifier, s.getType(tp), sc.Keyword)
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
	s.xmlFile.WriteString(str)
}

func (s *Service) indent() {
	s.indentation = s.indentation + "  "
}

func (s *Service) deindent() {
	if len(s.indentation) > 1 {
		s.indentation = s.indentation[:len(s.indentation)-2]
	}
}

func (s *Service) eat() tokenizer.Terminal {
	s.tokenizer.Advance()
	return s.tokenizer.ConsumeToken()
}

func (s *Service) eatBinOp() (tokenizer.Terminal, error) {
	return s.eatSymbol("+", "-", "*", "/", "&", "|", "<", ">", "=")
}

func (s *Service) eatSymbol(ss ...string) (tokenizer.Terminal, error) {
	s.tokenizer.Advance()

	if s.tokenizer.Token().IsSymbol(ss...) {
		return s.tokenizer.ConsumeToken(), nil
	}

	return tokenizer.Terminal{}, fmt.Errorf("expected one of symbols %v but token was %s", ss, s.tokenizer.Token())
}

func (s *Service) eatKeyword(ks ...string) (tokenizer.Terminal, error) {
	s.tokenizer.Advance()

	if s.tokenizer.Token().IsKeyword(ks...) {
		return s.tokenizer.ConsumeToken(), nil
	}

	return tokenizer.Terminal{}, fmt.Errorf("expected one of keywords %v but token was %s", ks, s.tokenizer.Token())
}

func (s *Service) eatIdentifier() (tokenizer.Terminal, error) {
	s.tokenizer.Advance()

	if s.tokenizer.Token().IsOfType(tokenizer.Identifier) {
		return s.tokenizer.ConsumeToken(), nil
	}

	return tokenizer.Terminal{}, fmt.Errorf("expected identifier but token was %s", s.tokenizer.Token())
}

func (s *Service) eatType(ts ...string) (tokenizer.Terminal, error) {
	s.tokenizer.Advance()

	if s.tokenizer.Token().IsOfType(tokenizer.Identifier) {
		return s.tokenizer.ConsumeToken(), nil
	}

	if s.tokenizer.Token().IsKeyword(ts...) {
		return s.tokenizer.ConsumeToken(), nil
	}

	return tokenizer.Terminal{}, fmt.Errorf("expected a type but token was %s", s.tokenizer.Token())
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
	return fmt.Sprintf("%s<%s> %s </%s>\n", s.indentation, tag, val, tag)
}

func (s *Service) openingTag(tag string) string {
	return s.indentation + "<" + tag + ">\n"
}

func (s *Service) closingTag(tag string) string {
	return s.indentation + "</" + tag + ">\n"
}
