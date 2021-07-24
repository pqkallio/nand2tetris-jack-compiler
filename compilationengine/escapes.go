package compilationengine

var xmlEscapes = map[string]string{
	"<":  "&lt;",
	">":  "&gt;",
	"\"": "&quot;",
	"&":  "&amp;",
}

func getEscapedSymbol(s string) string {
	if e, exists := xmlEscapes[s]; exists {
		return e
	}

	return s
}
