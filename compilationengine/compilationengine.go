package compilationengine

import (
	"fmt"
	"strconv"

	"github.com/pqkallio/nand2tetris-jack-compiler/symbols"
	"github.com/pqkallio/nand2tetris-jack-compiler/tokenizer"
	"github.com/pqkallio/nand2tetris-jack-compiler/vm"
)

type Service struct {
	tokenizer   *tokenizer.Service
	symbolTable *symbols.Table
	vmWriter    *vm.Writer
	className   string
}

func New(t *tokenizer.Service, vmWriter *vm.Writer) *Service {
	return &Service{t, symbols.New(), vmWriter, ""}
}

func (s *Service) Compile() error {
	t, err := s.eatKeyword("class")
	if err != nil {
		return err
	}

	return s.compileClass(t)
}

func (s *Service) compileClass(t tokenizer.Terminal) error {
	t, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	s.className = t.Identifier

	t, err = s.eatSymbol("{")
	if err != nil {
		return err
	}

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

	return nil
}

func (s *Service) compileSubroutineDec() error {
	tt, err := s.eatKeyword("constructor", "function", "method")
	if err != nil {
		return err
	}

	t, err := s.eatReturnType()
	if err != nil {
		return err
	}

	t, err = s.eatIdentifier()
	if err != nil {
		return err
	}

	funcName := t.Identifier

	s.symbolTable.SwitchSubroutineTo(t.Identifier)

	t, err = s.eatSymbol("(")
	if err != nil {
		return err
	}

	err = s.compileParameterList()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol(")")
	if err != nil {
		return err
	}

	err = s.compileSubroutineBody(funcName, tt.Keyword)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) compileSubroutineBody(funcName, funcType string) error {
	_, err := s.eatSymbol("{")
	if err != nil {
		return err
	}

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

	_, err = s.eatSymbol("}")
	if err != nil {
		return err
	}

	if funcType == "constructor" {
		s.vmWriter.WritePush(vm.Pointer, 0)
	}

	return nil
}

func (s *Service) compileStatements() error {
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

	return nil
}

func (s *Service) compileExpressionList() (uint, error) {
	nArgs := uint(0)

	_, err := s.eatSymbol(")")
	if err != nil {
		for {
			err = s.compileExpression()
			if err != nil {
				return 0, err
			}

			nArgs += 1

			_, err := s.eatSymbol(",")
			if err != nil {
				break
			}
		}
	} else {
		s.tokenizer.Rewind(0)
	}

	return nArgs, nil
}

func (s *Service) compileSubroutineCall() error {
	t, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	id := t.Identifier

	t, err = s.eatSymbol(".", "(")
	if err != nil {
		return err
	}

	switch sym := t.Symbol; sym {
	case ".":
		t, err = s.eatIdentifier()
		if err != nil {
			return err
		}

		id += "." + t.Identifier

		t, err = s.eatSymbol("(")
		if err != nil {
			return err
		}

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
	t := s.eat()

	switch tt := t.Type; tt {
	case tokenizer.IntegerConstant:
		i, _ := strconv.Atoi(t.IntegerConstant)
		s.vmWriter.WritePush(vm.Const, uint(i))
	case tokenizer.StringConstant:
		s.pushStringConstant(t.StringConstant)
	case tokenizer.Keyword:
		s.pushKeywordConstant(t.Keyword)
	case tokenizer.Identifier:
		t2, err := s.eatSymbol("(", "[", ".")
		if err != nil {
			e := s.symbolTable.Get(t.Identifier)

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
			e := s.symbolTable.Get(t.Identifier)

			s.vmWriter.WritePush(e.Scope.ToVMMemSeg(), e.Idx)

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
		}
	case tokenizer.Symbol:
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
		case "-":
			fallthrough
		case "~":
			err := s.compileTerm()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) compileExpression() error {
	err := s.compileTerm()
	if err != nil {
		return err
	}

	t, err := s.eatBinOp()
	if err == nil {
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

	return nil
}

func (s *Service) compileReturnStatement(t tokenizer.Terminal) error {
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

	s.vmWriter.WriteReturn()

	return nil
}

func (s *Service) compileDoStatement(t tokenizer.Terminal) error {
	err := s.compileSubroutineCall()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol(";")
	if err != nil {
		return err
	}

	s.vmWriter.WritePop(vm.Temp, 0)

	return nil
}

func (s *Service) compileWhileStatement(t tokenizer.Terminal) error {
	lblFalse := s.vmWriter.RegisterLabel("IF_FALSE")
	lblTrue := s.vmWriter.RegisterLabel("IF_TRUE")

	s.vmWriter.WriteLabel(lblTrue)

	t, err := s.eatSymbol("(")
	if err != nil {
		return err
	}

	err = s.compileExpression()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol(")")
	if err != nil {
		return err
	}

	s.vmWriter.WriteArithmetic(vm.Not)
	s.vmWriter.WriteIf(lblFalse)

	t, err = s.eatSymbol("{")
	if err != nil {
		return err
	}

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

	return nil
}

func (s *Service) compileIfStatement(t tokenizer.Terminal) error {
	lblFalse := s.vmWriter.RegisterLabel("IF_FALSE")
	lblTrue := s.vmWriter.RegisterLabel("IF_TRUE")

	t, err := s.eatSymbol("(")
	if err != nil {
		return err
	}

	err = s.compileExpression()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol(")")
	if err != nil {
		return err
	}

	s.vmWriter.WriteArithmetic(vm.Not)
	s.vmWriter.WriteIf(lblFalse)

	t, err = s.eatSymbol("{")
	if err != nil {
		return err
	}

	err = s.compileStatements()
	if err != nil {
		return err
	}

	t, err = s.eatSymbol("}")
	if err != nil {
		return err
	}

	s.vmWriter.WriteGoto(lblTrue)

	s.vmWriter.WriteLabel(lblFalse)

	t, err = s.eatKeyword("else")
	if err == nil {
		t, err = s.eatSymbol("{")
		if err != nil {
			return err
		}

		err = s.compileStatements()
		if err != nil {
			return err
		}

		t, err = s.eatSymbol("}")
		if err != nil {
			return err
		}
	}

	s.vmWriter.WriteLabel(lblTrue)

	return nil
}

func (s *Service) compileLetStatement(t tokenizer.Terminal) error {
	t, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	e := s.symbolTable.Get(t.Identifier)
	target := vm.MemEntry{e.Scope.ToVMMemSeg(), e.Idx}

	t, err = s.eatSymbol("[", "=")
	if err != nil {
		return nil
	}

	if t.Symbol == "[" {
		err = s.compileExpression()
		if err != nil {
			return err
		}

		t, err = s.eatSymbol("]")
		if err != nil {
			return err
		}

		t, err = s.eatSymbol("=")
		if err != nil {
			return err
		}

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

	return nil
}

func (s *Service) compileVarDec() error {
	t, err := s.eatKeyword("var")
	if err != nil {
		return err
	}

	tp, err := s.eatVarType()
	if err != nil {
		return err
	}

	id, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	s.symbolTable.Define(id.Identifier, s.getType(tp), "local")

	t, err = s.eatSymbol(",", ";")
	if err != nil {
		return err
	}

	for t.Symbol == "," {
		id, err = s.eatIdentifier()
		if err != nil {
			return err
		}

		s.symbolTable.Define(id.Identifier, s.getType(tp), "local")

		t, err = s.eatSymbol(",", ";")
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) compileParameterList() error {
	tp, err := s.eatVarType()
	if err != nil {
		return nil
	}

	id, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	s.symbolTable.Define(id.Identifier, s.getType(tp), "arg")

	for {
		_, err = s.eatSymbol(",")
		if err != nil {
			break
		}

		tp, err = s.eatVarType()
		if err != nil {
			return err
		}

		id, err = s.eatIdentifier()
		if err != nil {
			return err
		}

		s.symbolTable.Define(id.Identifier, s.getType(tp), "arg")
	}

	return nil
}

func (s *Service) compileClassVarDec() error {
	sc, err := s.eatKeyword("static", "field")
	if err != nil {
		return err
	}

	tp, err := s.eatVarType()
	if err != nil {
		return err
	}

	id, err := s.eatIdentifier()
	if err != nil {
		return err
	}

	s.symbolTable.Define(id.Identifier, s.getType(tp), sc.Keyword)

	t, err := s.eatSymbol(",", ";")
	if err != nil {
		return err
	}

	for t.Symbol != ";" {
		id, err = s.eatIdentifier()
		if err != nil {
			return err
		}

		s.symbolTable.Define(id.Identifier, s.getType(tp), sc.Keyword)

		t, err = s.eatSymbol(",", ";")
		if err != nil {
			return err
		}
	}

	return nil
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

func (s *Service) getType(t tokenizer.Terminal) string {
	if t.Type == tokenizer.Keyword {
		return t.Keyword
	}

	return t.Identifier
}
