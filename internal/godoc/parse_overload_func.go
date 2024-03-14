package godoc

import (
	"context"
	"go/ast"
	"go/doc"
	"golang.org/x/pkgsite/internal/log"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var pattern string = `__\d+`

// FindOverloadFuncThenAdd  Find overloaded functions and update function names
func FindOverloadFuncThenAdd(d *doc.Package) {
	if len(d.Funcs) == 0 {
		return
	}
	var overloadFuncName = make(map[string]string)
	for _, constO := range d.Consts {
		for _, name := range constO.Names {
			if strings.Contains(name, "Gopo_") {
				for _, spec := range constO.Decl.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range vs.Names {
							if strings.Contains(name.Name, "Gopo_") {
								n := strings.TrimPrefix(name.Name, "Gopo_")
								for _, v := range vs.Values {
									if bas, ok := v.(*ast.BasicLit); ok {
										// ignore error
										values, err := strconv.Unquote(bas.Value)
										if err != nil {
											continue
										}
										overloadFuncs := strings.Split(values, ",")
										for i2 := range overloadFuncs {
											overloadFuncName[overloadFuncs[i2]] = n
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	// even though overloadFuncName is empty, but the situation of funcName__N may occur, so it cannot be returned in advance
	var overloadFunc = make([]*doc.Func, 0, len(overloadFuncName))
	for _, funcO := range d.Funcs {
		match, err := isMatchName(funcO.Name)
		if err != nil {
			continue
		}
		if match {
			restoreName(funcO)
		}
		if name, ok := overloadFuncName[funcO.Name]; ok {
			newFunc := buildNewFunc(funcO, name)
			overloadFunc = append(overloadFunc, newFunc)
		}
	}
	for _, funs := range overloadFunc {
		d.Funcs = append(d.Funcs, funs)
	}

	sort.Slice(d.Funcs, func(i, j int) bool {
		return d.Funcs[i].Name < d.Funcs[j].Name
	})
}

// FindOverloadFuncTypeThenRestoreName func name: xxx_001 xxx_002
func FindOverloadFuncTypeThenRestoreName(types []*doc.Type) {
	for _, t := range types {
		for _, f := range t.Methods {
			match, err := isMatchName(f.Name)
			if err != nil {
				continue
			}
			if match {
				restoreName(f)
			}
		}
	}
}

func isMatchName(name string) (bool, error) {
	match, err := regexp.MatchString(pattern, name)
	if err != nil {
		log.Errorf(context.Background(), "match function name err", err.Error())
		return false, err
	}
	return match, nil
}

// restoreName restore overload func name
func restoreName(funcO *doc.Func) {
	re := regexp.MustCompile(pattern)
	name := re.ReplaceAllString(funcO.Name, "")
	funcO.Decl.Name.Name = name
	funcO.Name = name
}

func buildNewFunc(oldFunc *doc.Func, name string) *doc.Func {
	newFunc := &doc.Func{}
	newFunc.Doc = oldFunc.Doc
	newFunc.Recv = oldFunc.Recv
	newFunc.Orig = oldFunc.Orig
	newFunc.Decl = &ast.FuncDecl{}
	newFunc.Decl.Type = oldFunc.Decl.Type
	newFunc.Decl.Doc = oldFunc.Decl.Doc
	newFunc.Decl.Recv = oldFunc.Decl.Recv
	newFunc.Decl.Body = oldFunc.Decl.Body
	newFunc.Decl.Name = &ast.Ident{}
	newFunc.Decl.Name.NamePos = oldFunc.Decl.Name.NamePos
	newFunc.Decl.Name.Name = name
	newFunc.Decl.Name.Obj = oldFunc.Decl.Name.Obj
	newFunc.Level = oldFunc.Level
	newFunc.Examples = oldFunc.Examples
	newFunc.Name = name
	return newFunc
}
