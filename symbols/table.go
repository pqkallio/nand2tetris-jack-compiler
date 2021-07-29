package symbols

type Table struct {
	classTable      *table
	subroutineTable *table
}

var classScopes = []Scope{Static, Field}
var subroutineScopes = []Scope{Argument, Local}

func New() *Table {
	return &Table{
		newLocalTable("", classScopes...),
		newLocalTable("", subroutineScopes...),
	}
}

func (t *Table) Define(name, dataType, s string) *Entry {
	scope := Field

	switch s {
	case "local":
		scope = Local
	case "static":
		scope = Static
	case "arg":
		scope = Argument
	}

	if scope.In(classScopes...) {
		return t.classTable.Define(name, dataType, scope)
	}

	if scope.In(subroutineScopes...) {
		return t.subroutineTable.Define(name, dataType, scope)
	}

	return nil
}

func (t *Table) Get(name string) *Entry {
	var err error
	var e *Entry

	if e, err = t.subroutineTable.Get(name); err == nil {
		return e
	}

	e, _ = t.classTable.Get(name)

	return e
}

func (t *Table) SwitchSubroutineTo(subroutineName, funcType string) {
	t.subroutineTable = newLocalTable(funcType, subroutineScopes...)
}

func (t *Table) GetSymbolCount(scope Scope) uint {
	if scope.In(classScopes...) {
		return t.classTable.GetSymbolCount(scope)
	}

	return t.subroutineTable.GetSymbolCount(scope)
}
