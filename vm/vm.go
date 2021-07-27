package vm

import (
	"fmt"
	"os"
)

type (
	Op     string
	MemSeg string
	Writer struct {
		out    *os.File
		lblIdx uint
	}
)

const (
	Add Op = "add"
	Sub Op = "sub"
	Eq  Op = "eq"
	Gt  Op = "gt"
	Lt  Op = "lt"
	And Op = "and"
	Or  Op = "or"
	Neg Op = "neg"
	Not Op = "not"

	Const   MemSeg = "constant"
	Arg     MemSeg = "argument"
	Local   MemSeg = "local"
	Static  MemSeg = "static"
	This    MemSeg = "this"
	That    MemSeg = "that"
	Pointer MemSeg = "pointer"
	Temp    MemSeg = "temp"
)

type MemEntry struct {
	Seg MemSeg
	Idx uint
}

func New(out *os.File) *Writer {
	return &Writer{out, 0}
}

func (w *Writer) WritePush(seg MemSeg, idx uint) {
	w.writeLine(fmt.Sprintf("push %s %d", string(seg), idx))
}

func (w *Writer) WritePop(seg MemSeg, idx uint) {
	w.writeLine(fmt.Sprintf("pop %s %d", string(seg), idx))
}

func (w *Writer) WriteArithmetic(op Op) {
	w.writeLine(string(op))
}

func (w *Writer) WriteLabel(lbl string) {
	w.writeLine(fmt.Sprintf("label %s", lbl))
}

func (w *Writer) WriteGoto(lbl string) {
	w.writeLine(fmt.Sprintf("goto %s", lbl))
}

func (w *Writer) WriteIf(lbl string) {
	w.writeLine(fmt.Sprintf("if-goto %s", lbl))
}

func (w *Writer) WriteCall(name string, nArgs uint) {
	w.writeLine(fmt.Sprintf("call %s %d", name, nArgs))
}

func (w *Writer) WriteFunc(name string, nLocals uint) {
	w.writeLine(fmt.Sprintf("function %s %d", name, nLocals))
}

func (w *Writer) WriteReturn() {
	w.writeLine("return")
}

func (w *Writer) writeLine(s string) {
	w.out.WriteString(s + "\n")
}

func (w *Writer) RegisterLabel(lbl string) string {
	idx := w.lblIdx

	w.lblIdx += 1

	return fmt.Sprintf("%s%d", lbl, idx)
}
