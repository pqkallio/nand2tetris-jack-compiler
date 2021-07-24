package tokenizer

var symbols = "{}()[].,;+-*&|<>=~"
var commentStarters = "*/"

type keywords []string

func (kws keywords) Contains(s string) bool {
	for _, s2 := range kws {
		if s2 == s {
			return true
		}
	}

	return false
}

var kws = keywords{
	"class",
	"method",
	"function",
	"constructor",
	"int",
	"boolean",
	"char",
	"void",
	"var",
	"static",
	"field",
	"let",
	"do",
	"if",
	"else",
	"while",
	"return",
	"true",
	"false",
	"null",
	"this",
}
