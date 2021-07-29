package symbols

import "fmt"

type table struct {
	symbols map[string]*Entry
	idxs    map[Scope]uint
	scopes  []Scope
}

func newLocalTable(funcType string, scopes ...Scope) *table {
	idxs := map[Scope]uint{}
	if funcType == "method" {
		idxs[Argument] = 1
	}

	return &table{map[string]*Entry{}, idxs, scopes}
}

func (l *table) nextIdxFor(scope Scope) uint {
	idx, exists := l.idxs[scope]

	if !exists {
		idx = 0
	}

	l.idxs[scope] = idx + 1

	return idx
}

func (l *table) Define(name, dataType string, scope Scope) *Entry {
	if _, exists := l.symbols[name]; exists {
		return nil
	}

	if !scope.In(l.scopes...) {
		return nil
	}

	idx := l.nextIdxFor(scope)

	e := Entry{name, scope, dataType, idx}

	l.symbols[name] = &e

	return &e
}

func (l *table) Get(name string) (*Entry, error) {
	e, exists := l.symbols[name]
	if !exists {
		return nil, fmt.Errorf("%s not in symbol table", name)
	}

	return e, nil
}

func (t *table) GetSymbolCount(scope Scope) uint {
	if n, exists := t.idxs[scope]; exists {
		return n
	}

	return 0
}
