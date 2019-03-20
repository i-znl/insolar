/*
 *    Copyright 2019 Insolar Technologies
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"strings"
)

type Parser struct {
	sourceFilename string
	sourceCode     []byte
	sourceNode     *ast.File
	generator      *Generator
	Package        string
	StateMachines  []*stateMachine
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func exitWithError(err string, a ...interface{}) {
	fmt.Println(fmt.Sprintf(err, a...))
	os.Exit(1)
}

func (p *Parser) readStateMachinesInterfaceFile() {
	var err error
	p.sourceCode, err = ioutil.ReadFile(p.sourceFilename)
	checkErr(err)
	fSet := token.NewFileSet()
	p.sourceNode, err = parser.ParseFile(fSet, p.sourceFilename, nil, parser.ParseComments)
	checkErr(err)
	p.Package = p.sourceNode.Name.Name
}

func (p *Parser) findEachStateMachine() {
	for _, decl := range p.sourceNode.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			currType, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			currStruct, ok := currType.Type.(*ast.InterfaceType)
			if !ok || !isStateMachineTag(genDecl.Doc) {
				continue
			}

			machine := &stateMachine{
				Name:    currType.Name.Name,
				Package: p.Package,
				States:  []state{{Name: "Init"}},
			}
			p.parseStateMachineInterface(machine, currStruct)
		}
	}
}

func isStateMachineTag(group *ast.CommentGroup) bool {
	for _, comment := range group.List {
		if strings.Contains(comment.Text, "conveyor: state_machine") {
			return true
		}
	}
	return false
}

func getFieldTypes(code []byte, fieldList *ast.FieldList) []string {
	if fieldList == nil {
		return make([]string, 0)
	}
	result := make([]string, len(fieldList.List))
	for i, field := range fieldList.List {
		fieldType := field.Type
		result[i] = string(code[fieldType.Pos()-1 : fieldType.End()-1])
	}
	return result
}

func (p *Parser) parseStateMachineInterface(machine *stateMachine, source *ast.InterfaceType) {
	curPos := token.Pos(0)
	for _, methodItem := range source.Methods.List {
		if len(methodItem.Names) == 0 {
			continue
		}
		if methodItem.Pos() <= curPos {
			exitWithError("Incorrect order of methods")
		}
		curPos = methodItem.Pos()
		methodType := methodItem.Type.(*ast.FuncType)

		currentHandler := &handler{
			machine: machine,
			state:   len(machine.States) - 1,
			Name:    methodItem.Names[0].Name,
			Params:  getFieldTypes(p.sourceCode, methodType.Params),
			Results: getFieldTypes(p.sourceCode, methodType.Results),
		}

		switch {
		case currentHandler.Name == "GetTypeID":
		case strings.HasPrefix(currentHandler.Name, "state"):
			currentHandler.setAsState()
		case strings.HasPrefix(currentHandler.Name, "initPresent"):
			currentHandler.setAsInit()
		case strings.HasPrefix(currentHandler.Name, "initFuture"):
			currentHandler.setAsInitFuture()
		case strings.HasPrefix(currentHandler.Name, "initPast"):
			currentHandler.setAsInitPast()
		case strings.HasPrefix(currentHandler.Name, "errorPresent"):
			currentHandler.setAsErrorState()
		case strings.HasPrefix(currentHandler.Name, "errorFuture"):
			currentHandler.setAsErrorStateFuture()
		case strings.HasPrefix(currentHandler.Name, "errorPast"):
			currentHandler.setAsErrorStatePast()
		case strings.HasPrefix(currentHandler.Name, "migrateFromPresent"):
			currentHandler.setAsMigration()
		case strings.HasPrefix(currentHandler.Name, "migrateFromFuture"):
			currentHandler.setAsMigrationFuturePresent()
		case strings.HasPrefix(currentHandler.Name, "transitPresent"):
			currentHandler.setAsTransition()
		case strings.HasPrefix(currentHandler.Name, "transitFuture"):
			currentHandler.setAsTransitionFuture()
		case strings.HasPrefix(currentHandler.Name, "transitPast"):
			currentHandler.setAsTransitionPast()
		case strings.HasPrefix(currentHandler.Name, "responsePresent"):
			currentHandler.setAsAdapterResponse()
		case strings.HasPrefix(currentHandler.Name, "responseFuture"):
			currentHandler.setAsAdapterResponseFuture()
		case strings.HasPrefix(currentHandler.Name, "responsePast"):
			currentHandler.setAsAdapterResponsePast()
		case strings.HasPrefix(currentHandler.Name, "errorResponsePresent"):
			currentHandler.setAsAdapterResponseError()
		case strings.HasPrefix(currentHandler.Name, "errorResponseFuture"):
			currentHandler.setAsAdapterResponseErrorFuture()
		case strings.HasPrefix(currentHandler.Name, "errorResponsePast"):
			currentHandler.setAsAdapterResponseErrorPast()
		default:
			exitWithError("Unknown handler: %s", currentHandler.Name)
		}
	}
	p.StateMachines = append(p.StateMachines, machine)
	p.generator.stateMachines = append(p.generator.stateMachines, machine)
}