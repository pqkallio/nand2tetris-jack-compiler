package tokenizer

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

type Service struct {
	f  *os.File
	ts []Terminal
	tp int
	b  []byte
	c  bool
}

func New(f *os.File) *Service {
	return &Service{
		f,
		[]Terminal{},
		-1,
		make([]byte, 1),
		false,
	}
}

func (t *Service) Token() Terminal {
	return t.ts[t.tp]
}

func (t *Service) Advance() {
	if len(t.ts) == 0 || t.c {
		if t.tp != len(t.ts)-1 {
			t.tp += 1
		} else {
			tk := t.readNextToken()
			t.addToken(tk)
		}
		t.c = false
	}
}

func (t *Service) ConsumeToken() Terminal {
	t.c = true
	return t.ts[t.tp]
}

func (s *Service) addToken(t Terminal) {
	s.ts = append(s.ts, t)
	s.tp += 1
}

func (s *Service) Rewind(nSteps int) error {
	if nSteps > s.tp {
		return fmt.Errorf("too many steps to go back")
	}

	s.tp -= nSteps
	s.c = false

	return nil
}

func (t *Service) readNextToken() Terminal {
	s := ""

	for {
		if _, err := t.f.Read(t.b); err != nil {
			if errors.Is(err, io.EOF) {
				return Terminal{Type: EOF}
			}

			return Terminal{Type: Error}
		}

		if unicode.IsSpace(rune(t.b[0])) {
			continue
		}

		s = string(t.b[0])

		if s == "/" {
			t2 := t.parseSlash()

			if t2.Type == Comment {
				continue
			}

			return t2
		}

		if strings.Contains(symbols, s) {
			return Terminal{Type: Symbol, Symbol: s}
		}

		if t.b[0] > 0x2f && t.b[0] < 0x3a {
			return t.parseInteger(s)
		}

		if s == "\"" {
			return t.parseString()
		}

		return t.parseIdentifier(s)
	}
}

func (t *Service) parseSlash() Terminal {
	baseCase := Terminal{Type: Symbol, Symbol: "/"}

	if n, err := t.f.Read(t.b); err != nil || n == 0 {
		return baseCase
	}

	if unicode.IsSpace(rune(t.b[0])) {
		return baseCase
	}

	s := string(t.b[0])

	if strings.Contains(commentStarters, s) {
		t.skipComment(s)
		return Terminal{Type: Comment}
	}

	t.f.Seek(-1, 1)
	return baseCase
}

func (t *Service) skipComment(start string) {
	switch start {
	case "*":
		t.skipMultilineComment()
	case "/":
		t.skipSingleLineComment()
	}
}

func (t *Service) skipMultilineComment() {
	starHit := false

	for {
		if n, err := t.f.Read(t.b); err != nil || n == 0 {
			return
		}

		s := string(t.b[0])

		switch s {
		case "*":
			starHit = true
		case "/":
			if starHit {
				return
			}
		default:
			starHit = false
		}
	}
}

func (t *Service) skipSingleLineComment() {
	for {
		if n, err := t.f.Read(t.b); err != nil || n == 0 {
			return
		}

		if string(t.b[0]) == "\n" {
			return
		}
	}
}

func (t *Service) parseInteger(s string) Terminal {
	for {
		if n, err := t.f.Read(t.b); err != nil || n == 0 {
			return Terminal{Type: Error}
		}

		if unicode.IsSpace(rune(t.b[0])) {
			break
		}

		if t.b[0] < 0x30 || t.b[0] > 0x39 {
			t.f.Seek(-1, 1)
			break
		}

		s = s + string(t.b[0])
	}

	return Terminal{Type: IntegerConstant, IntegerConstant: s}
}

func (t *Service) parseString() Terminal {
	s := ""

	for {
		if n, err := t.f.Read(t.b); err != nil || n == 0 {
			return Terminal{Type: Error}
		}

		s2 := string(t.b[0])

		if s2 == "\"" {
			break
		}

		s = s + s2
	}

	return Terminal{Type: StringConstant, StringConstant: s}
}

func (t *Service) parseIdentifier(s string) Terminal {
	for {
		if n, err := t.f.Read(t.b); err != nil || n == 0 {
			return Terminal{Type: Error}
		}

		if unicode.IsSpace(rune(t.b[0])) {
			break
		}

		s2 := string(t.b[0])

		if strings.Contains(symbols, s2) {
			t.f.Seek(-1, 1)
			break
		}

		s = s + s2
	}

	if kws.Contains(s) {
		return Terminal{Type: Keyword, Keyword: s}
	}

	return Terminal{Type: Identifier, Identifier: s}
}
