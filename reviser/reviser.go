package reviser

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/incu6us/goimport-reviser/helper"
)

func Execute(projectName, filePath string) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		log.Println(err)
		return nil, errors.WithStack(err)
	}

	imports := combineAllImports(f)

	stdImports, projectImports, generalImports := groupImports(projectName, imports)

	fixImports(f, stdImports, generalImports, projectImports)

	out, err := generateFile(fset, f)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return out, nil
}

func groupImports(projectName string, imports []string) ([]string, []string, []string) {
	var (
		stdImports     []string
		projectImports []string
		generalImports []string
	)

	for _, imprt := range imports {
		if _, ok := helper.StdPackages[imprt]; ok {
			stdImports = append(stdImports, imprt)
			continue
		}

		if strings.Contains(imprt, projectName) {
			projectImports = append(projectImports, imprt)
			continue
		}

		generalImports = append(generalImports, imprt)
	}

	sort.Strings(stdImports)
	sort.Strings(generalImports)
	sort.Strings(projectImports)

	return stdImports, projectImports, generalImports
}

func generateFile(fset *token.FileSet, file *ast.File) ([]byte, error) {
	var output []byte
	buffer := bytes.NewBuffer(output)
	if err := printer.Fprint(buffer, fset, file); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func fixImports(f *ast.File, stdImports []string, generalImports []string, projectImports []string) {
	for _, decl := range f.Decls {
		switch decl.(type) {
		case *ast.GenDecl:
			dd := decl.(*ast.GenDecl)
			if dd.Tok == token.IMPORT {
				var specs []ast.Spec

				linesCounter := len(stdImports)
				for _, stdImport := range stdImports {
					iSpec := &ast.ImportSpec{Path: &ast.BasicLit{Value: stdImport}}
					specs = append(specs, iSpec)

					linesCounter--

					if linesCounter == 0 && len(generalImports) > 0 {
						iSpec = &ast.ImportSpec{Path: &ast.BasicLit{Value: ""}}

						specs = append(specs, iSpec)
					}
				}

				linesCounter = len(generalImports)
				for _, generalImport := range generalImports {
					iSpec := &ast.ImportSpec{Path: &ast.BasicLit{Value: generalImport}}
					specs = append(specs, iSpec)

					linesCounter--

					if linesCounter == 0 && len(generalImports) > 0 {
						iSpec = &ast.ImportSpec{Path: &ast.BasicLit{Value: ""}}

						specs = append(specs, iSpec)
					}
				}

				for _, projectImport := range projectImports {
					iSpec := &ast.ImportSpec{Path: &ast.BasicLit{Value: projectImport}}
					specs = append(specs, iSpec)
				}

				dd.Specs = specs
			}
		}
	}
}

func combineAllImports(f *ast.File) []string {
	var imports []string

	for _, decl := range f.Decls {
		switch decl.(type) {
		case *ast.GenDecl:
			dd := decl.(*ast.GenDecl)
			if dd.Tok == token.IMPORT {
				for _, spec := range dd.Specs {
					var importSpecStr string
					importSpec := spec.(*ast.ImportSpec)

					if importSpec.Name != nil {
						importSpecStr = strings.Join([]string{importSpec.Name.String(), importSpec.Path.Value}, " ")
					} else {
						importSpecStr = importSpec.Path.Value
					}

					imports = append(imports, importSpecStr)
				}
			}
		}
	}

	return imports
}
