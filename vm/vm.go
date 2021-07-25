package vm

import "os"

type (
	Op     string
	MemSeg string
	Writer struct {
		out *os.File
	}
)

const (
	Add Op = "ADD"
	Sub Op = "SUB"
	Eq  Op = "EQ"
	Gt  Op = "GT"
	Lt  Op = "LT"
	And Op = "AND"
	Or  Op = "OR"
	Neg Op = "NEG"
	Not Op = "NOT"

	Const   MemSeg = "CONST"
	Arg     MemSeg = "ARG"
	Local   MemSeg = "LOCAL"
	Static  MemSeg = "STATIC"
	This    MemSeg = "THIS"
	That    MemSeg = "THAT"
	Pointer MemSeg = "POINTER"
	Temp    MemSeg = "TEMP"
)

func New(out *os.File) *Writer {
	return &Writer{out}
}

func (w *Writer) WritePush(seg MemSeg, idx uint) {

}

func (w *Writer) WritePop(seg MemSeg, idx uint) {

}

func (w *Writer) WriteArithmetic(op Op) {

}

func (w *Writer) WriteLabel(lbl string) {

}

func (w *Writer) WriteGoto(lbl string) {

}

func (w *Writer) WriteIf(lbl string) {

}

func (w *Writer) WriteCall(name string, nArgs uint) {

}

func (w *Writer) WriteFunc(name string, nLocals uint) {

}

func (w *Writer) WriteReturn() {

}
