/*
Copyright © 2021 Sentry

Generate static documentation of Job definitions
*/
package cmd

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/getsentry/go-load-tester/utils"
)

const DocFileName = "docs/TestFormat.md"
const WritingTestsDocFileName = "docs/Writing-tests.md"

type FieldDefinition struct {
	FieldName     string
	FieldType     string
	Documentation string
}

type StructDefinition struct {
	TypeName      string
	Documentation string
	Fields        []FieldDefinition
}

type GeneralDeclaration struct {
	Name          string
	Documentation string
	Source        string
}

// structFilter used to filter the structures returned (true means type will not be filtered)
type structFilter func(typeSpec *ast.TypeSpec, structSpec *ast.StructType, typeDoc string) bool

var makeDocsParams struct {
	sourceDirectory string
}

// master runs the load tester in master mode.
var makeDocs = &cobra.Command{
	Use:   "update-docs",
	Short: "Extract docs from source code into static files.",
	Long:  `Creates static documents in the docs subdirectory.`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Info().Msgf("Creating documents in docs directory and README")
		updateTestDocument()
		updateReadme()
		updateWriteTestDocument()
	},
}

func updateWriteTestDocument() {
	templateRaw, err := os.ReadFile("_Writing-tests-template.md")
	if err != nil {
		log.Error().Err(err).Msg("Could not generate documentation, error reading _Writing-tests-template.md.")
		return
	}
	parsedTemplate, err := template.New("template").Parse(string(templateRaw))

	_ = os.Mkdir("docs", os.ModePerm)
	writeTestsFile, err := os.Create(WritingTestsDocFileName)
	if err != nil {
		log.Error().Err(err).Msg("Could not generate documentation, error creating README.md file.")
		return
	}
	defer func() { _ = writeTestsFile.Close() }()

	documentedTypes := []string{"RegisterTestType", "LoadSplitter", "SimpleLoadSplitter", "LoadTesterBuilder"}

	params := getDocForTypes("tests/main.go", documentedTypes)

	err = parsedTemplate.Execute(writeTestsFile, params)
	if err != nil {
		log.Error().Err(err).Msg("Could not generate documentation, error parsing template file.")
		return
	}
}

func getDocForTypes(fileName string, typeNames []string) map[string]string {
	fset := token.NewFileSet()
	// var retVal []StructDefinition
	src, err := os.ReadFile(fileName)
	if err != nil {
		log.Error().Err(err).Msgf("Could not read file %s", fileName)
		return nil
	}
	parsedFile, err := parser.ParseFile(fset, fileName, src, parser.ParseComments)
	if err != nil {
		log.Error().Err(err).Msgf("Could not parse file %s", fileName)
		return nil
	}

	declTemplate := declarationTemplate()
	retVal := make(map[string]string, len(typeNames))

	for _, val := range typeNames {
		retVal[val] = "temp"
	}

	for _, decl := range parsedFile.Decls {
		var declaration GeneralDeclaration
		switch d := decl.(type) {
		case *ast.FuncDecl:
			declaration = getFunctionDeclaration(src, d)
		case *ast.GenDecl:
			declaration = getTypeDeclaration(src, d)
		}

		// check if the name is needed
		if _, ok := retVal[declaration.Name]; ok {
			buf := new(bytes.Buffer)
			err = declTemplate.Execute(buf, declaration)
			retVal[declaration.Name] = buf.String()
		}
	}

	return retVal
}

func declarationTemplate() *template.Template {
	tmplString := `
## {{.Name}}

{{.Documentation}}
~~~go
{{.Source}}
~~~
`

	tmpl, err := template.New("funcDoc").Parse(tmplString)
	if err != nil {
		log.Error().Err(err).Msg("Could not parse embedded template for declaration.")
		panic(err)
	}
	return tmpl
}

func getTypeDeclaration(src []byte, decl *ast.GenDecl) GeneralDeclaration {
	name := ""
	if decl.Tok == token.TYPE && decl.Specs != nil && len(decl.Specs) == 1 {
		typeSpec, ok := decl.Specs[0].(*ast.TypeSpec)
		if ok {
			name = typeSpec.Name.Name
		}
	}

	return GeneralDeclaration{
		Name:          name,
		Documentation: getDoc(decl.Doc, false),
		Source:        string(src[decl.Pos()-1 : decl.End()]),
	}
}

func getFunctionDeclaration(src []byte, decl *ast.FuncDecl) GeneralDeclaration {
	return GeneralDeclaration{
		Name:          decl.Name.Name,
		Documentation: getDoc(decl.Doc, false),
		Source:        string(src[decl.Type.Pos()-1 : decl.Type.End()]),
	}
}

func updateReadme() {
	templateRaw, err := os.ReadFile("README-template.md")
	if err != nil {
		log.Error().Err(err).Msg("Could not generate documentation, error reading README-template.md.")

		return
	}
	parsedTemplate, err := template.New("template").Parse(string(templateRaw))
	usage := rootCmd.UsageString()
	runUsage := runCmd.UsageString()
	workerUsage := workerCmd.UsageString()
	masterUsage := masterCmd.UsageString()

	readmeFile, err := os.Create("README.md")
	if err != nil {
		log.Error().Err(err).Msg("Could not generate documentation, error creating README.md file.")
		return
	}
	defer func() { _ = readmeFile.Close() }()

	params := struct {
		RootUsage   string
		RunUsage    string
		WorkerUsage string
		MasterUsage string
	}{
		RootUsage:   usage,
		RunUsage:    runUsage,
		WorkerUsage: workerUsage,
		MasterUsage: masterUsage,
	}

	err = parsedTemplate.Execute(readmeFile, params)
	if err != nil {
		log.Error().Err(err).Msg("Could not generate documentation, error parsing template file.")
		return
	}
}

func updateTestDocument() {
	log.Info().Msg("Generating documents...")
	dir, files, err := getCodeFiles()

	if err != nil {
		log.Error().Err(err).Msg("Error trying to find source files")
	}

	jobDefinitions, err := extractStructDefinitions(dir, filterJobsAndHelperTypes, files)

	if err != nil {
		log.Error().Err(err).Msg("Error trying to extract structure definitions from code")
	}
	_ = os.Mkdir("docs", os.ModePerm)
	f, err := os.Create(DocFileName)
	defer func() { _ = f.Close() }()
	err = writeTemplate(f, jobDefinitions)

	if err != nil {
		log.Error().Err(err).Msg("Error generating documentation from template")
	}
}

func writeTemplate(outStream io.Writer, jobDefinitions []StructDefinition) error {
	templateName := "_jobs.template.md"
	t := template.New("jobs")
	src, err := os.ReadFile(templateName)
	if err != nil {
		log.Error().Err(err).Msgf("Could not read file %s", templateName)
		return err
	}
	t, err = t.Parse(string(src))
	if err != nil {
		log.Error().Err(err).Msgf("Could not read file %s", templateName)
		return err
	}

	data := struct{ Tests []StructDefinition }{
		Tests: jobDefinitions,
	}

	err = t.Execute(outStream, data)
	if err != nil {
		log.Error().Err(err).Msgf("Error executing template %s", templateName)
	}

	return nil
}

func filterJobsAndHelperTypes(typeSpec *ast.TypeSpec, structSpec *ast.StructType, typeDoc string) bool {
	if typeSpec == nil || typeSpec.Name == nil || structSpec == nil {
		return false
	}
	if strings.Contains(typeSpec.Name.Name, "Job") &&
		!strings.Contains(typeSpec.Name.Name, "Raw") {
		return true
	}
	// look for @doc annotations
	annotation := getDocAnnotation(typeDoc)
	if annotation != nil {
		scope := annotation["scope"]
		if scope == "job" {
			return true
		}
	}
	return false
}

func extractStructDefinitions(directory string, filter structFilter, fileEntries []os.DirEntry) ([]StructDefinition, error) {
	fset := token.NewFileSet()
	var retVal []StructDefinition
	// parsed files
	for _, fileEntry := range fileEntries {
		filePath := filepath.Join(directory, fileEntry.Name())
		src, err := os.ReadFile(filePath)
		if err != nil {
			log.Error().Err(err).Msgf("Could not read file %s", fileEntry.Name())
			continue
		}
		parsedFile, err := parser.ParseFile(fset, fileEntry.Name(), src, parser.ParseComments)
		if err != nil {
			log.Error().Err(err).Msgf("Could not parse file %s", fileEntry.Name())
			continue
		}
		// look at all top level entries
		for _, decl := range parsedFile.Decls {
			typeSpec, ok := getStructTypeSpec(decl, filter)
			if ok {
				retVal = append(retVal, *typeSpec)
			}
		}
	}
	return retVal, nil
}

// getTypeSpec returns the TypeSpec from a Decl if that Decl is a GenDecl of a type
// if not it returns nil
func getStructTypeSpec(decl ast.Decl, filter structFilter) (*StructDefinition, bool) {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return nil, false
	}
	if genDecl.Tok == token.TYPE && genDecl.Specs != nil && len(genDecl.Specs) == 1 {
		doc := getDoc(genDecl.Doc, false)
		typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
		if ok {
			structType, ok := typeSpec.Type.(*ast.StructType)
			if ok {
				if !filter(typeSpec, structType, doc) {
					return nil, false
				}
				var fieldsDoc []FieldDefinition = nil
				if structType.Fields != nil && structType.Fields.List != nil {
					fieldsDoc = make([]FieldDefinition, 0, len(structType.Fields.List))
					for _, field := range structType.Fields.List {
						fieldDef := getFieldDefinition(field)
						if fieldDef != nil {
							fieldsDoc = append(fieldsDoc, *fieldDef)
						}
					}
				}

				return &StructDefinition{
					TypeName:      typeSpec.Name.Name,
					Documentation: removeDocAnnotation(doc),
					Fields:        fieldsDoc,
				}, true
			}

		}
	}
	return nil, false
}

func getFieldDefinition(field *ast.Field) *FieldDefinition {
	if field == nil {
		return nil
	}

	fieldDoc := getDoc(field.Doc, true)
	fieldName := ""
	if field.Names != nil && len(field.Names) >= 1 {
		fieldName = field.Names[0].Name
	}

	retVal := FieldDefinition{
		FieldName:     utils.LowerFirstLetter(fieldName),
		Documentation: utils.LowerFirstLetter(strings.TrimSpace(fieldDoc)),
	}

	return &retVal
}

func getDoc(comments *ast.CommentGroup, noNewLine bool) string {
	sep := "\n"
	if noNewLine {
		sep = " "
	}

	if comments != nil && comments.List != nil {
		allComments := make([]string, 0, len(comments.List))
		for _, comment := range comments.List {
			text := comment.Text
			if strings.HasPrefix(text, "//") {
				// line comment
				allComments = append(allComments, text[2:]+sep)
			} else {
				// block style comment
				allComments = append(allComments, text)
			}
		}
		retVal := strings.Join(allComments, "")
		if noNewLine {
			strings.ReplaceAll(retVal, "\n", " ")
		}
		return retVal
	}
	return ""
}

// getCodeFiles returns a list of DirEntry for all the go files at inside the tests directory
// no recursive search in subdirectories is attempted
func getCodeFiles() (string, []os.DirEntry, error) {
	var err error
	dirName := "."
	if len(makeDocsParams.sourceDirectory) != 0 {
		dirName = makeDocsParams.sourceDirectory
	}
	testsFileDir := filepath.Join(dirName, "tests")

	testsFileDir, err = filepath.Abs(testsFileDir)
	if err != nil {
		log.Error().Err(err).Msgf("Error trying to read directory %s", dirName)
		return "", nil, err
	}

	var files []os.DirEntry
	files, err = os.ReadDir(testsFileDir)

	if err != nil {
		log.Error().Err(err).Msgf("Error trying to read directory %s", dirName)
		return "", nil, err
	}

	var retVal = make([]os.DirEntry, 0, len(files)/2)
	for _, file := range files {
		if !file.IsDir() {
			if strings.HasSuffix(file.Name(), ".go") {
				retVal = append(retVal, file)
			}
		}
	}
	return testsFileDir, retVal, nil
}

var docAnnotation *regexp.Regexp

func getDocAnnotation(doc string) map[string]string {
	var retVal map[string]string
	s := docAnnotation.FindString(doc)
	if len(s) >= 6 {
		annotationString := s[5 : len(s)-1]
		// try to figure out if we have a dictionary
		if strings.Contains(annotationString, "{") {
			err := json.Unmarshal([]byte(annotationString), &retVal)
			if err != nil {
				log.Error().Err(err).Msg("Failed to deserialize doc dictionary")
				return nil // an error occured trying to deserialize a
			}
			return retVal
		}
		// only some string remove " at beginning and end (if exists)
		if strings.HasPrefix(annotationString, "\"") {
			annotationString = annotationString[1:]
		}
		if strings.HasSuffix(annotationString, "\"") {
			annotationString = annotationString[:len(annotationString)-1]
		}
		retVal = make(map[string]string)
		retVal["data"] = annotationString
	}

	return retVal
}

// cleans up string of doc annotations so that they do not appear in documentation.
func removeDocAnnotation(doc string) string {
	for {
		location := docAnnotation.FindStringIndex(doc)
		if location == nil {
			return doc // no doc found return the input string
		}
		if location[0] == 0 {
			// the doc annotation is at the beginning just return the string after
			doc = doc[location[1]:]

		} else if location[1] == len(location) {
			// the doc annotation is at the end, just return the beginning
			doc = doc[:location[0]]

		} else {
			start := doc[:location[0]]
			end := doc[location[1]:]
			doc = start + end
		}

	}

}

func init() {
	docAnnotation, _ = regexp.Compile(`@doc\([^()]*\)`)
	makeDocs.Flags().StringVar(&makeDocsParams.sourceDirectory, "source-dir", "", "directory for source files (omit if current directory)")
	rootCmd.AddCommand(makeDocs)
}
