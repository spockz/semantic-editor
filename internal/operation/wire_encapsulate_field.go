// Package operation exposes one typed, fail-closed Go field encapsulation through CLI, MCP, receipts, and batches.
// WHY: Accessor generation and every reference rewrite must share one module snapshot and one buffer application.
package operation

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/spockz/semantic-editor/internal/astedit"
	"github.com/spockz/semantic-editor/internal/backend"
	"github.com/spockz/semantic-editor/internal/pipeline"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"golang.org/x/tools/go/packages"
)

const (
	encapsulateFieldToolName  = "refactor_encapsulate_field"
	encapsulateFieldOperation = "encapsulate_field"
)

type encapsulateFieldResult struct {
	Edit    FileEditRes
	Draft   EditReceiptDraft
	Resolve ContinuationResolver
	Receipt EditReceipt
	Root    string
}

func encapsulateFieldDef() Def[map[string]any, encapsulateFieldResult] {
	params := make([]ParameterContract, 0, 8+len(readProjectParams))
	params = append(params,
		structuralStringParam("file", "Target: existing Go file that declares the selected private field.", true, true, RoleTarget),
		structuralStringParam("symbol", "Target: qualified existing field identity in Type.Field form.", true, true, RoleTarget),
		structuralStringParam("getter", "Replacement: fresh private method name for the generated getter.", true, false, RoleValue),
		structuralStringParam("setter", "Replacement: fresh private method name for the generated setter.", true, false, RoleValue),
		structuralStringParam(contractDestinationFileField, "Destination: existing same-package Go file for both generated accessors; defaults to file.", false, true, RoleTarget),
		ParameterContract{Name: "access_modifier", JSONName: "access_modifier", CLIName: "access-modifier", Type: ParamString, Role: RolePolicy, Description: "Policy: private accessors only in this release.", Enums: []string{"private"}, Default: "private"},
		encapsulateFieldImportPolicyParam(),
		structuralStringParam("expected_revision", "Precondition: optional full-file revision for the field declaration file.", false, true, RoleTarget),
	)
	params = append(params, readProjectParams...)
	target := SymbolTargetSchema().Properties
	outputSchema := ValueSchema{Kind: SchemaObject, Properties: map[string]ValueSchema{
		"result": {Kind: SchemaString}, "file": {Kind: SchemaString}, "symbol": {Kind: SchemaString}, contractDestinationFileField: {Kind: SchemaString}, "diff": {Kind: SchemaString}, editReceiptField: EditReceiptSchema(),
	}, Required: []string{"result", editReceiptField}}
	output := OperationOutputContract{ResultSchema: &outputSchema, Continuations: []OutputContinuation{
		{Path: editReceiptContinuationFilePath, Schema: target["file"], Role: RoleTarget, Consumers: []string{semanticInspectSymbolToolName, semanticRenameToolName, semanticReplaceBodyToolName}},
		{Path: editReceiptContinuationSymbolPath, Schema: target["symbol"], Role: RoleTarget, Consumers: []string{semanticInspectSymbolToolName, semanticRenameToolName, semanticReplaceBodyToolName}},
		{Path: editReceiptContinuationRevisionPath, Schema: target["expected_revision"], Role: RoleTarget, Consumers: []string{semanticInspectSymbolToolName, semanticRenameToolName, semanticReplaceBodyToolName}},
	}}
	parse := func(raw map[string]any) (map[string]any, error) {
		if err := CheckParams(raw, params); err != nil {
			return nil, err
		}
		result := maps.Clone(raw)
		destination, err := ParseString(raw, contractDestinationFileField, "destination-file", false)
		if err != nil {
			return nil, err
		}
		if destination == "" {
			destination = stringParam(raw, "file")
		}
		result[contractDestinationFileField] = destination
		expectedRevision, err := ParseString(raw, wireExpectedRevision, wireCLIExpectedRevision, false)
		if err != nil {
			return nil, err
		}
		if expectedRevision != "" {
			result[wireExpectedRevision] = expectedRevision
		}
		accessModifier, err := parseEncapsulateAccessModifier(raw)
		if err != nil {
			return nil, err
		}
		result["access_modifier"] = accessModifier
		result[wireAutoOrganizeImports], err = ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, true)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return Def[map[string]any, encapsulateFieldResult]{
		Key: canonicalEncapsulateFieldKey, Summary: "Use this tool instead of generating accessors and replacing field references separately. It encapsulates one supported private Go struct field, preserves storage, and rewrites only typed direct reads and standalone assignments whose evaluation order is proven.",
		Params: params, Level: LevelWorkspace, CLIName: "refactor-encapsulate-field", MCPName: encapsulateFieldToolName,
		Batchable: true, BufferedBatch: true, ExampleRaw: map[string]any{"file": "config.go", "symbol": "config.timeout", "getter": "timeoutValue", "setter": "setTimeoutValue"}, Output: &output,
		Parse: parse, Handlers: map[backend.LanguageID]func(context.Context, CallContext, map[string]any) (encapsulateFieldResult, error){backend.LanguageGo: runEncapsulateField},
		TargetPreflight: func(ctx context.Context, cc CallContext, raw map[string]any) error {
			_, _, _, err := planEncapsulateField(ctx, cc, raw)
			return err
		},
		Format: func(result encapsulateFieldResult) (string, error) { return formatFileEdit(result.Edit) },
		FinalizeOutput: func(cc CallContext, _ map[string]any, result encapsulateFieldResult) (encapsulateFieldResult, error) {
			receipt, err := finalizeCallReceipt(cc.Ctx, cc, result.Root, result.Draft, result.Resolve)
			if err != nil {
				return encapsulateFieldResult{}, fmt.Errorf("finalize field encapsulation receipt: %w", err)
			}
			result.Receipt = receipt
			return result, nil
		},
		ProjectOutput: func(result encapsulateFieldResult) (any, error) {
			projected := map[string]any{"result": result.Edit.Detail, editReceiptField: result.Receipt}
			if result.Edit.Diff != "" {
				projected["diff"] = result.Edit.Diff
			}
			return projected, nil
		},
	}
}

func encapsulateFieldImportPolicyParam() ParameterContract {
	return ParameterContract{Name: wireAutoOrganizeImports, JSONName: wireAutoOrganizeImports, CLIName: wireCLIAutoOrganizeImports, Type: ParamBoolean, Role: RolePolicy, Description: "Automatically add required imports for generated accessor types (default true). Required imports are included in the validated candidate even in a batch; the batch-level option controls final import cleanup.", Default: true}
}

func parseEncapsulateAccessModifier(raw map[string]any) (string, error) {
	modifier, err := ParseString(raw, "access_modifier", "access-modifier", false)
	if err != nil {
		return "", err
	}
	if modifier == "" {
		modifier = "private"
	}
	return modifier, nil
}

func runEncapsulateField(ctx context.Context, cc CallContext, raw map[string]any) (encapsulateFieldResult, error) {
	snapshot, intent, plan, err := planEncapsulateField(ctx, cc, raw)
	if err != nil {
		return encapsulateFieldResult{}, err
	}
	readSet, err := signatureReadSet(cc.WorkDir, snapshot)
	if err != nil {
		return encapsulateFieldResult{}, err
	}
	planned := make(map[string][]byte, len(plan.Files))
	draft := EditReceiptDraft{Operation: encapsulateFieldOperation, Status: EditPersisted, ReadSet: readSet}
	for _, item := range plan.Files {
		planned[item.File] = item.After
		relative, relErr := filepath.Rel(cc.WorkDir, item.File)
		if relErr != nil || !filepath.IsLocal(relative) {
			return encapsulateFieldResult{}, fmt.Errorf("planned file %q is outside workspace; run field encapsulation from the module root", item.File)
		}
		draft.PlannedChanges = append(draft.PlannedChanges, FileRevision{Path: filepath.ToSlash(relative), Revision: structuralRevision(item.After)})
	}
	destination := resolveWorkPath(cc.WorkDir, stringParam(raw, contractDestinationFileField))
	if destinationSource, ok := snapshot.Sources[destination]; !ok {
		return encapsulateFieldResult{}, fmt.Errorf("accessor destination %q was not captured", destination)
	} else if _, plannedDestination := planned[destination]; !plannedDestination {
		planned[destination] = destinationSource
	}
	destinationRelative, relErr := filepath.Rel(cc.WorkDir, destination)
	if relErr != nil || !filepath.IsLocal(destinationRelative) {
		return encapsulateFieldResult{}, fmt.Errorf("accessor destination %q is outside workspace; run field encapsulation from the module root", destination)
	}
	revision := structuralRevision(planned[destination])
	draft.Continuations = []SymbolTarget{
		{File: filepath.ToSlash(destinationRelative), Symbol: "(*" + intent.ownerType + ")." + intent.getter, ExpectedRevision: &revision},
		{File: filepath.ToSlash(destinationRelative), Symbol: "(*" + intent.ownerType + ")." + intent.setter, ExpectedRevision: &revision},
	}
	finishDelta := surroundingDelta(snapshot.Ctx, cc.WorkDir, cc.DeferVerification)
	changed, diffs, applyErr := applySignaturePlan(snapshot, plan)
	if applyErr != nil && !changed {
		return encapsulateFieldResult{}, applyErr
	}
	delta, diagnosticStatus := finishDelta()
	selectedPath := resolveWorkPath(cc.WorkDir, stringParam(raw, "file"))
	detail := "Generated field accessors and rewrote every supported typed reference."
	if !changed {
		detail = "The field already has the requested accessors and references."
	}
	edit := FileEditRes{File: selectedPath, Display: displayWorkPath(cc.WorkDir, stringParam(raw, "file")), Symbol: stringParam(raw, "symbol"), Detail: detail, Diff: strings.Join(diffs, "\n"), Delta: delta, DiagnosticStatus: diagnosticStatus, HasDelta: true}
	if applyErr != nil {
		edit.Detail = "Field encapsulation publication did not complete: " + applyErr.Error()
	}
	if !changed {
		draft.Status = EditNoOp
		draft.PlannedChanges = nil
		draft.Continuations = nil
	} else if diagnosticStatus == "changed" {
		draft.Status = EditPersistedWithDiagnostics
	}
	resolve := func(resolveCtx context.Context, candidate SymbolTarget) (SymbolTarget, bool, error) {
		return resolveEncapsulatedMethod(resolveCtx, cc.WorkDir, candidate)
	}
	return encapsulateFieldResult{Edit: edit, Draft: draft, Resolve: resolve, Root: cc.WorkDir}, nil
}

type encapsulateFieldIntent struct{ ownerType, fieldName, fieldType, getter, setter string }

func planEncapsulateField(ctx context.Context, cc CallContext, raw map[string]any) (*structuralSnapshot, encapsulateFieldIntent, astedit.SignaturePlan, error) {
	snapshot, selected, err := prepareEncapsulateFieldSnapshot(ctx, cc, raw)
	if err != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, err
	}
	if expected := stringParam(raw, "expected_revision"); expected != "" && expected != structuralRevision(snapshot.Sources[selected]) {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, fmt.Errorf("field declaration source revision changed")
	}
	owner, named, ownerName, field, err := resolveEncapsulateFieldOwner(snapshot, stringParam(raw, "symbol"))
	if err != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, err
	}
	if scopeErr := ensureEncapsulateCallerScope(cc.WorkDir, snapshot, owner); scopeErr != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, scopeErr
	}
	fieldName := field.Name()
	if checkErr := structuralFieldSourceSet(snapshot, owner, named, ownerName, fieldName); checkErr != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, checkErr
	}
	if checkErr := refuseTypedPositionalOwnerLiterals(snapshot, owner, named); checkErr != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, checkErr
	}
	if checkErr := ensureEncapsulateVariantCoverage(snapshot, owner, named); checkErr != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, checkErr
	}
	if checkErr := refuseEncapsulateReflectionOrUnsafe(snapshot, owner); checkErr != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, checkErr
	}
	getter, setter := stringParam(raw, "getter"), stringParam(raw, "setter")
	if checkErr := validateEncapsulateAccessorNames(snapshot, owner, named, getter, setter); checkErr != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, checkErr
	}
	destinationRelative := stringParam(raw, contractDestinationFileField)
	if destinationRelative == "" {
		destinationRelative = stringParam(raw, "file")
	}
	destination := resolveWorkPath(cc.WorkDir, destinationRelative)
	if _, ok := snapshot.Sources[destination]; !ok || !samePackageFile(owner, destination) {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, fmt.Errorf("destination_file must be an existing source file in the selected Go package")
	}
	autoOrganizeImports, err := ParseBool(raw, wireAutoOrganizeImports, wireCLIAutoOrganizeImports, true)
	if err != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, err
	}
	fieldType, requiredImports, err := encapsulateFieldTypeString(owner, destination, field.Type(), autoOrganizeImports)
	if err != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, err
	}
	intent := encapsulateFieldIntent{ownerType: ownerName, fieldName: fieldName, fieldType: fieldType, getter: getter, setter: setter}
	rewrites, err := collectEncapsulateFieldRewrites(snapshot, owner, owner.Fset.PositionFor(named.Obj().Pos(), false), field, getter, setter)
	if err != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, err
	}
	files := make([]astedit.SignatureSourceFile, 0, len(snapshot.Sources))
	for path, source := range snapshot.Sources {
		if filepath.Ext(path) == ".go" && filepath.IsLocal(strings.TrimPrefix(path, snapshot.Root+string(filepath.Separator))) {
			files = append(files, astedit.SignatureSourceFile{File: path, Source: source})
		}
	}
	plan, err := astedit.PlanEncapsulateField(files, astedit.EncapsulateFieldIntent{DestinationFile: destination, OwnerType: ownerName, FieldName: fieldName, FieldType: fieldType, Getter: getter, Setter: setter}, rewrites)
	if err != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, err
	}
	if err := addEncapsulateRequiredImports(plan, destination, requiredImports); err != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, err
	}
	if err := validateSignatureOverlay(snapshot, plan); err != nil {
		return nil, encapsulateFieldIntent{}, astedit.SignaturePlan{}, fmt.Errorf("validate encapsulated field overlay: %w", err)
	}
	return snapshot, intent, plan, nil
}

func addEncapsulateRequiredImports(plan astedit.SignaturePlan, destination string, required []string) error {
	if len(required) == 0 {
		return nil
	}
	for index := range plan.Files {
		if plan.Files[index].File != destination {
			continue
		}
		after, err := pipeline.AddExplicitImports(plan.Files[index].File, plan.Files[index].After, required)
		if err != nil {
			return fmt.Errorf("add required accessor type imports: %w", err)
		}
		plan.Files[index].After = after
		return nil
	}
	return fmt.Errorf("accessor destination %q is missing from the encapsulation plan", destination)
}

func ensureEncapsulateCallerScope(workDir string, snapshot *structuralSnapshot, owner *packages.Package) error {
	callerRoot, err := filepath.Abs(workDir)
	if err != nil {
		return fmt.Errorf("resolve field encapsulation caller root %q: %w", workDir, err)
	}
	if filepath.Clean(callerRoot) != filepath.Clean(snapshot.Root) {
		return fmt.Errorf("field encapsulation requires the Go module root as caller workspace")
	}
	for _, pkg := range snapshot.Packages {
		if pkg.Types == nil || pkg.Types.Path() != owner.Types.Path() || pkg.Name != owner.Name {
			continue
		}
		for _, sourceFile := range pkg.CompiledGoFiles {
			absolute, err := filepath.Abs(sourceFile)
			if err != nil {
				return fmt.Errorf("resolve owner package source %q: %w", sourceFile, err)
			}
			relative, err := filepath.Rel(workDir, absolute)
			if err != nil || !filepath.IsLocal(relative) {
				return fmt.Errorf("owner package source %q is outside workspace; run field encapsulation from the module root", sourceFile)
			}
		}
	}
	return nil
}

func prepareEncapsulateFieldSnapshot(ctx context.Context, cc CallContext, raw map[string]any) (*structuralSnapshot, string, error) {
	selected := resolveWorkPath(cc.WorkDir, stringParam(raw, "file"))
	root, err := structuralModuleRoot(selected)
	if err != nil {
		return nil, "", err
	}
	snapshot, err := prepareStructuralSnapshot(ctx, cc, stringParam(raw, "file"))
	if err != nil {
		return nil, "", err
	}
	if observeErr := sourcebuf.ObserveTree(snapshot.Ctx, root, ".go"); observeErr != nil {
		return nil, "", fmt.Errorf("observe Go module source membership: %w", observeErr)
	}
	loaded, sources, err := loadStructuralPackages(snapshot.Ctx, root, nil)
	if err != nil {
		return nil, "", err
	}
	snapshot.Packages, snapshot.Sources = loaded, sources
	if err := snapshot.Buffer.AssertDisk(); err != nil {
		return nil, "", fmt.Errorf("revalidate Go module membership before field planning: %w", err)
	}
	return snapshot, selected, nil
}

func resolveEncapsulateFieldOwner(snapshot *structuralSnapshot, symbol string) (*packages.Package, *types.Named, string, *types.Var, error) {
	parts := strings.Split(symbol, ".")
	if len(parts) != 2 || !token.IsIdentifier(parts[1]) || token.Lookup(parts[1]).IsKeyword() {
		return nil, nil, "", nil, fmt.Errorf("symbol must be one qualified Type.Field identity")
	}
	parsedOwner, err := parseStructuralFieldSelector(parts[0])
	if err != nil {
		return nil, nil, "", nil, err
	}
	ownerSelector := parsedOwner.TypeName
	if parsedOwner.PackageName != "" {
		ownerSelector = parsedOwner.PackageName + "." + parsedOwner.TypeName
	}
	owner, named, ownerName, err := structuralFieldOwner(snapshot, ownerSelector)
	if err != nil {
		return nil, nil, "", nil, err
	}
	if ast.IsExported(ownerName) || named.TypeParams().Len() != 0 {
		return nil, nil, "", nil, fmt.Errorf("encapsulate field requires an unexported non-generic owner type")
	}
	structure := named.Underlying().(*types.Struct)
	for field := range structure.Fields() {
		if field.Embedded() {
			return nil, nil, "", nil, fmt.Errorf("encapsulate field refuses structs with embedded fields")
		}
	}
	for field := range structure.Fields() {
		if field.Name() == parts[1] && !field.Exported() {
			return owner, named, ownerName, field, nil
		}
	}
	return nil, nil, "", nil, fmt.Errorf("encapsulate field requires one existing unexported direct field")
}

func refuseEncapsulateReflectionOrUnsafe(snapshot *structuralSnapshot, owner *packages.Package) error {
	for _, pkg := range snapshot.Packages {
		if pkg.Types == nil || pkg.Types.Path() != owner.Types.Path() || pkg.Name != owner.Name {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, spec := range file.Imports {
				path, err := strconv.Unquote(spec.Path.Value)
				if err == nil && (path == "reflect" || path == "unsafe") {
					return fmt.Errorf("encapsulate field refuses package %s because it imports %s", pkg.PkgPath, path)
				}
			}
		}
	}
	return nil
}

func refuseTypedPositionalOwnerLiterals(snapshot *structuralSnapshot, owner *packages.Package, named *types.Named) error {
	for _, pkg := range snapshot.Packages {
		if pkg.Types == nil || pkg.Types.Path() != owner.Types.Path() || pkg.Name != owner.Name || pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			var positional string
			ast.Inspect(file, func(node ast.Node) bool {
				literal, ok := node.(*ast.CompositeLit)
				if !ok || len(literal.Elts) == 0 {
					return true
				}
				literalType := encapsulateNamed(pkg.TypesInfo.TypeOf(literal))
				if literalType == nil || !sameGoObjectPosition(pkg.Fset, literalType.Obj(), owner.Fset.PositionFor(named.Obj().Pos(), false)) {
					return true
				}
				for _, element := range literal.Elts {
					if _, keyed := element.(*ast.KeyValueExpr); !keyed {
						positional = pkg.Fset.PositionFor(literal.Pos(), false).Filename
						return false
					}
				}
				return true
			})
			if positional != "" {
				return fmt.Errorf("encapsulate field refuses positional owner literal in %q", positional)
			}
		}
	}
	return nil
}

func validateEncapsulateAccessorNames(snapshot *structuralSnapshot, owner *packages.Package, named *types.Named, getter, setter string) error {
	if !isPrivateEncapsulateMethod(getter) || !isPrivateEncapsulateMethod(setter) || getter == setter {
		return fmt.Errorf("getter and setter must be distinct private Go identifiers")
	}
	methodSet := types.NewMethodSet(types.NewPointer(named))
	for _, name := range []string{getter, setter} {
		if methodSet.Lookup(owner.Types, name) != nil {
			return fmt.Errorf("accessor method %q already exists on the selected owner", name)
		}
	}
	for _, pkg := range snapshot.Packages {
		if pkg.Types == nil || pkg.Types.Path() != owner.Types.Path() || pkg.Name != owner.Name || pkg.TypesInfo == nil {
			continue
		}
		for _, syntax := range pkg.Syntax {
			if err := scanEncapsulateSyntax(pkg, syntax, owner, named, getter, setter); err != nil {
				return err
			}
		}
	}
	return nil
}

func scanEncapsulateSyntax(pkg *packages.Package, syntax *ast.File, owner *packages.Package, named *types.Named, getter, setter string) error {
	var scanErr error
	ast.Inspect(syntax, func(node ast.Node) bool {
		if scanErr != nil {
			return false
		}
		switch typed := node.(type) {
		case *ast.InterfaceType:
			scanErr = rejectEncapsulateInterfaceName(pkg, typed, getter, setter)
		case *ast.StructType:
			scanErr = rejectEmbeddedEncapsulateOwner(pkg, typed, owner, named)
		}
		return scanErr == nil
	})
	return scanErr
}

func rejectEncapsulateInterfaceName(pkg *packages.Package, syntax *ast.InterfaceType, getter, setter string) error {
	value, ok := pkg.TypesInfo.TypeOf(syntax).(*types.Interface)
	if !ok {
		return nil
	}
	value.Complete()
	for method := range value.Methods() {
		if name := method.Name(); name == getter || name == setter {
			return fmt.Errorf("accessor name %q occurs in an interface method set in package %s", name, pkg.PkgPath)
		}
	}
	return nil
}

func rejectEmbeddedEncapsulateOwner(pkg *packages.Package, syntax *ast.StructType, owner *packages.Package, named *types.Named) error {
	for _, field := range syntax.Fields.List {
		if len(field.Names) != 0 {
			continue
		}
		ownerNamed := encapsulateNamed(pkg.TypesInfo.TypeOf(field.Type))
		if sameNamedGoObject(pkg.Fset, ownerNamed, owner.Fset.PositionFor(named.Obj().Pos(), false)) {
			return fmt.Errorf("encapsulate field refuses an owner embedded elsewhere in its package")
		}
	}
	return nil
}

func encapsulateFieldTypeString(owner *packages.Package, destination string, fieldType types.Type, allowAdd bool) (string, []string, error) {
	imports := make([]string, 0)
	aliases := make(map[string]string)
	reserved := encapsulateDestinationNames(owner, destination)
	text := types.TypeString(fieldType, func(pkg *types.Package) string {
		name := encapsulateImportQualifier(owner, destination, pkg)
		if pkg.Path() == owner.Types.Path() || name != "" {
			return name
		}
		if alias, exists := aliases[pkg.Path()]; exists {
			return alias
		}
		if !allowAdd {
			aliases[pkg.Path()] = ""
			return ""
		}
		alias := encapsulateFreshImportAlias(pkg, reserved)
		aliases[pkg.Path()] = alias
		reserved[alias] = true
		imports = append(imports, alias+" "+strconv.Quote(pkg.Path()))
		return alias
	})
	for path, alias := range aliases {
		if alias == "" {
			return "", nil, fmt.Errorf("accessor destination %q does not import field type package %q with a usable alias and auto_organize_imports is false", destination, path)
		}
	}
	slices.Sort(imports)
	return text, imports, nil
}

func encapsulateFreshImportAlias(target *types.Package, used map[string]bool) string {
	base := target.Name()
	if !token.IsIdentifier(base) || token.Lookup(base).IsKeyword() {
		base = filepath.Base(target.Path())
	}
	if !token.IsIdentifier(base) || token.Lookup(base).IsKeyword() {
		base = "pkg"
	}
	return nextEncapsulateAlias(base, used)
}

func encapsulateDestinationNames(owner *packages.Package, destination string) map[string]bool {
	used := make(map[string]bool)
	for _, name := range types.Universe.Names() {
		used[name] = true
	}
	for _, name := range owner.Types.Scope().Names() {
		used[name] = true
	}
	for _, syntax := range owner.Syntax {
		path, err := filepath.Abs(owner.Fset.PositionFor(syntax.Pos(), false).Filename)
		if err != nil || path != destination {
			continue
		}
		for _, spec := range syntax.Imports {
			var name string
			if spec.Name != nil {
				name = spec.Name.Name
			} else {
				importPath, unquoteErr := strconv.Unquote(spec.Path.Value)
				if unquoteErr == nil {
					for _, imported := range owner.Imports {
						if imported.PkgPath == importPath {
							name = imported.Name
							break
						}
					}
				}
			}
			if name != "" {
				used[name] = true
			}
		}
	}
	return used
}

func nextEncapsulateAlias(base string, used map[string]bool) string {
	for alias, suffix := base, 2; ; suffix++ {
		if !used[alias] {
			return alias
		}
		alias = fmt.Sprintf("%s%d", base, suffix)
	}
}

func encapsulateImportQualifier(owner *packages.Package, destination string, target *types.Package) string {
	if target.Path() == owner.Types.Path() {
		return ""
	}
	for _, syntax := range owner.Syntax {
		path, err := filepath.Abs(owner.Fset.PositionFor(syntax.Pos(), false).Filename)
		if err != nil || path != destination {
			continue
		}
		for _, spec := range syntax.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil || importPath != target.Path() {
				continue
			}
			var object types.Object
			if spec.Name == nil {
				object = owner.TypesInfo.Implicits[spec]
			} else {
				object = owner.TypesInfo.Defs[spec.Name]
			}
			packageName, ok := object.(*types.PkgName)
			if !ok || packageName.Name() == "_" || packageName.Name() == "." {
				return ""
			}
			return packageName.Name()
		}
	}
	return ""
}

func isPrivateEncapsulateMethod(name string) bool {
	return token.IsIdentifier(name) && !token.Lookup(name).IsKeyword() && !token.IsExported(name)
}

func encapsulateNamed(typ types.Type) *types.Named {
	if typ == nil {
		return nil
	}
	typ = types.Unalias(typ)
	if pointer, ok := typ.(*types.Pointer); ok {
		typ = types.Unalias(pointer.Elem())
	}
	named, _ := typ.(*types.Named)
	return named
}

func sameNamedGoObject(fset *token.FileSet, named *types.Named, expected token.Position) bool {
	if named == nil || named.Obj() == nil {
		return false
	}
	actual := fset.PositionFor(named.Obj().Pos(), false)
	actualPath, actualErr := filepath.Abs(actual.Filename)
	expectedPath, expectedErr := filepath.Abs(expected.Filename)
	return actualErr == nil && expectedErr == nil && actualPath == expectedPath && actual.Offset == expected.Offset
}

func ensureEncapsulateVariantCoverage(snapshot *structuralSnapshot, owner *packages.Package, named *types.Named) error {
	directory := filepath.Dir(owner.Fset.PositionFor(named.Obj().Pos(), false).Filename)
	active := make(map[string]bool)
	for _, pkg := range snapshot.Packages {
		for _, filename := range pkg.CompiledGoFiles {
			absolute, err := filepath.Abs(filename)
			if err == nil && filepath.Dir(absolute) == directory {
				active[absolute] = true
			}
		}
		if pkg.Types == nil || pkg.Types.Path() != owner.Types.Path() || pkg.Name != owner.Name {
			continue
		}
		if slices.ContainsFunc(pkg.Syntax, ast.IsGenerated) {
			return fmt.Errorf("encapsulate field refuses generated Go source in the selected package")
		}
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("inspect Go build variants in %q: %w", directory, err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		path, err := filepath.Abs(filepath.Join(directory, entry.Name()))
		if err != nil {
			return fmt.Errorf("resolve Go build variant %q: %w", entry.Name(), err)
		}
		if !active[path] {
			return fmt.Errorf("encapsulate field cannot prove inactive or excluded Go source %q", entry.Name())
		}
	}
	_ = snapshot
	return nil
}

func samePackageFile(pkg *packages.Package, path string) bool {
	for _, file := range pkg.CompiledGoFiles {
		absolute, err := filepath.Abs(file)
		if err == nil && absolute == path {
			return true
		}
	}
	return false
}

func collectEncapsulateFieldRewrites(snapshot *structuralSnapshot, owner *packages.Package, ownerPosition token.Position, field *types.Var, getter, setter string) ([]astedit.FieldReferenceRewrite, error) {
	fieldPosition := owner.Fset.PositionFor(field.Pos(), false)
	fieldFile, err := filepath.Abs(fieldPosition.Filename)
	if err != nil {
		return nil, fmt.Errorf("resolve selected field declaration path: %w", err)
	}
	rewrites := make(map[string]astedit.FieldReferenceRewrite)
	seenPackage := false
	for _, pkg := range snapshot.Packages {
		if pkg.Types == nil || pkg.Types.Path() != owner.Types.Path() || pkg.Name != owner.Name {
			continue
		}
		seenPackage = true
		if err := collectEncapsulatePackageRewrites(snapshot, pkg, ownerPosition, field, fieldPosition, fieldFile, getter, setter, rewrites); err != nil {
			return nil, err
		}
	}
	if !seenPackage {
		return nil, fmt.Errorf("selected field package variants were not completely analyzed")
	}
	result := make([]astedit.FieldReferenceRewrite, 0, len(rewrites))
	for _, rewrite := range rewrites {
		result = append(result, rewrite)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].File == result[j].File {
			return result[i].Start < result[j].Start
		}
		return result[i].File < result[j].File
	})
	return result, nil
}

func collectEncapsulatePackageRewrites(snapshot *structuralSnapshot, pkg *packages.Package, ownerPosition token.Position, field *types.Var, fieldPosition token.Position, fieldFile, getter, setter string, rewrites map[string]astedit.FieldReferenceRewrite) error {
	if pkg.TypesInfo == nil || pkg.Fset == nil || len(pkg.Errors) != 0 || pkg.IllTyped {
		return fmt.Errorf("cannot prove every field reference in package variant %s", pkg.ID)
	}
	for _, syntax := range pkg.Syntax {
		filename, err := filepath.Abs(pkg.Fset.PositionFor(syntax.Pos(), false).Filename)
		if err != nil {
			return fmt.Errorf("resolve field reference source: %w", err)
		}
		if filepath.Dir(filename) != filepath.Dir(fieldFile) {
			continue
		}
		if _, ok := snapshot.Sources[filename]; !ok {
			return fmt.Errorf("field reference source %q was not captured", filename)
		}
		if err := collectEncapsulateFileRewrites(pkg, syntax, filename, snapshot.Sources[filename], ownerPosition, field, fieldPosition, getter, setter, rewrites); err != nil {
			return err
		}
	}
	return nil
}

func collectEncapsulateFileRewrites(pkg *packages.Package, syntax *ast.File, filename string, source []byte, ownerPosition token.Position, field *types.Var, fieldPosition token.Position, getter, setter string, rewrites map[string]astedit.FieldReferenceRewrite) error {
	parent := encapsulateParents(syntax)
	for identifier, object := range pkg.TypesInfo.Uses {
		identifierFile, err := filepath.Abs(pkg.Fset.PositionFor(identifier.Pos(), false).Filename)
		if err != nil || identifierFile != filename || !sameGoObjectPosition(pkg.Fset, object, fieldPosition) {
			continue
		}
		if err := collectEncapsulateIdentifier(pkg, syntax, source, parent, identifier, ownerPosition, field, fieldPosition, getter, setter, rewrites); err != nil {
			return err
		}
	}
	return nil
}

func collectEncapsulateIdentifier(pkg *packages.Package, syntax *ast.File, source []byte, parent map[ast.Node]ast.Node, identifier *ast.Ident, ownerPosition token.Position, field *types.Var, fieldPosition token.Position, getter, setter string, rewrites map[string]astedit.FieldReferenceRewrite) error {
	selector, ok := parent[identifier].(*ast.SelectorExpr)
	if ok && selector.Sel == identifier {
		selection := pkg.TypesInfo.Selections[selector]
		if selection == nil || !sameGoObjectPosition(pkg.Fset, selection.Obj(), fieldPosition) {
			return fmt.Errorf("field reference is not a direct typed selection")
		}
		if len(selection.Index()) != 1 {
			return fmt.Errorf("encapsulate field refuses promoted field selections")
		}
		return collectOneEncapsulateRewrite(pkg, syntax, source, selector, parent, ownerPosition, fieldPosition, getter, setter, rewrites)
	}
	if keyedLiteralField(parent, identifier) {
		return nil
	}
	return fmt.Errorf("encapsulate field found an unsupported reference to %q", field.Name())
}

func collectOneEncapsulateRewrite(pkg *packages.Package, file *ast.File, source []byte, selector *ast.SelectorExpr, parent map[ast.Node]ast.Node, ownerPosition, fieldPosition token.Position, getter, setter string, rewrites map[string]astedit.FieldReferenceRewrite) error {
	receiver, ok := selector.X.(*ast.Ident)
	if !ok {
		return fmt.Errorf("encapsulate field refuses side-effecting, nested, or indexed receiver expressions")
	}
	selection := pkg.TypesInfo.Selections[selector]
	named := encapsulateNamed(selection.Recv())
	if named == nil || !sameNamedGoObject(pkg.Fset, named, ownerPosition) {
		return fmt.Errorf("encapsulate field refuses references through a different named receiver type")
	}
	var rewrite astedit.FieldReferenceRewrite
	switch container := parent[selector].(type) {
	case *ast.AssignStmt:
		assignmentRewrite, skip, err := encapsulateAssignmentSelectorRewrite(pkg, file, source, parent, container, selector, receiver, ownerPosition, fieldPosition, getter, setter)
		if err != nil {
			return err
		}
		if skip {
			return nil
		}
		rewrite = assignmentRewrite
	case *ast.ValueSpec:
		if len(container.Names) != 1 || len(container.Values) != 1 || container.Values[0] != selector {
			return fmt.Errorf("encapsulate field read must be the complete initializer of one local")
		}
		getterRewrite, err := selectorGetterRewrite(pkg, file, selector, getter)
		if err != nil {
			return err
		}
		rewrite = getterRewrite
	case *ast.ReturnStmt:
		if len(container.Results) != 1 || container.Results[0] != selector {
			return fmt.Errorf("encapsulate field read must be the sole returned expression")
		}
		getterRewrite, err := selectorGetterRewrite(pkg, file, selector, getter)
		if err != nil {
			return err
		}
		rewrite = getterRewrite
	default:
		return fmt.Errorf("encapsulate field found a read or write outside a supported statement")
	}
	if rewrite.File == "" {
		rewrite.File = pkg.Fset.PositionFor(file.Pos(), false).Filename
	}
	absolute, err := filepath.Abs(rewrite.File)
	if err != nil {
		return fmt.Errorf("resolve field rewrite path: %w", err)
	}
	rewrite.File = absolute
	key := fmt.Sprintf("%s:%d:%d", rewrite.File, rewrite.Start, rewrite.End)
	if prior, exists := rewrites[key]; exists && prior.Replacement != rewrite.Replacement {
		return fmt.Errorf("conflicting typed reference rewrites at %s", key)
	}
	rewrites[key] = rewrite
	return nil
}

func encapsulateAssignmentSelectorRewrite(pkg *packages.Package, file *ast.File, source []byte, parent map[ast.Node]ast.Node, assignment *ast.AssignStmt, selector *ast.SelectorExpr, receiver *ast.Ident, ownerPosition, fieldPosition token.Position, getter, setter string) (astedit.FieldReferenceRewrite, bool, error) {
	if len(assignment.Lhs) > 0 && assignment.Lhs[0] == selector {
		rewrite, err := encapsulateSetterRewrite(pkg, file, source, parent, assignment, receiver, ownerPosition, fieldPosition, getter, setter)
		return rewrite, false, err
	}
	if len(assignment.Lhs) == 1 && len(assignment.Rhs) == 1 && assignment.Rhs[0] == selector && (assignment.Tok == token.DEFINE || assignment.Tok == token.ASSIGN) {
		if encapsulateAssignmentWritesField(pkg, assignment, fieldPosition) {
			return astedit.FieldReferenceRewrite{}, true, nil
		}
		rewrite, err := selectorGetterRewrite(pkg, file, selector, getter)
		return rewrite, false, err
	}
	return astedit.FieldReferenceRewrite{}, false, fmt.Errorf("encapsulate field read must be the complete value of one assignment")
}

func encapsulateSetterRewrite(pkg *packages.Package, file *ast.File, source []byte, parent map[ast.Node]ast.Node, assignment *ast.AssignStmt, receiver *ast.Ident, ownerPosition, fieldPosition token.Position, getter, setter string) (astedit.FieldReferenceRewrite, error) {
	if len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 || assignment.Tok != token.ASSIGN {
		return astedit.FieldReferenceRewrite{}, fmt.Errorf("encapsulate field supports only one-target standalone assignments")
	}
	if !encapsulateStatementListPosition(parent, assignment) {
		return astedit.FieldReferenceRewrite{}, fmt.Errorf("encapsulate field refuses assignments in statement headers")
	}
	start := pkg.Fset.PositionFor(assignment.Pos(), false).Offset
	end := pkg.Fset.PositionFor(assignment.End(), false).Offset
	if hasCommentInRange(pkg.Fset, file, start, end) {
		return astedit.FieldReferenceRewrite{}, fmt.Errorf("encapsulate field refuses comments inside rewritten assignments")
	}
	receiverArgument, err := encapsulateSetterReceiver(pkg, receiver, ownerPosition)
	if err != nil {
		return astedit.FieldReferenceRewrite{}, err
	}
	value := assignment.Rhs[0]
	if rhsSelector, ok := value.(*ast.SelectorExpr); ok {
		selection := pkg.TypesInfo.Selections[rhsSelector]
		if selection != nil && sameGoObjectPosition(pkg.Fset, selection.Obj(), fieldPosition) {
			rhsReceiver, ok := rhsSelector.X.(*ast.Ident)
			rhsNamed := encapsulateNamed(selection.Recv())
			if !ok || len(selection.Index()) != 1 || rhsNamed == nil || !sameNamedGoObject(pkg.Fset, rhsNamed, ownerPosition) {
				return astedit.FieldReferenceRewrite{}, fmt.Errorf("encapsulate field refuses a nested or promoted right-hand-side field read")
			}
			value = &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent(rhsReceiver.Name), Sel: ast.NewIdent(getter)}}
		}
	}
	call := encapsulateSetterAssignment(receiverArgument, setter, value)
	var formatted bytes.Buffer
	if err := format.Node(&formatted, pkg.Fset, call); err != nil {
		return astedit.FieldReferenceRewrite{}, fmt.Errorf("format field setter call: %w", err)
	}
	return astedit.FieldReferenceRewrite{File: pkg.Fset.PositionFor(file.Pos(), false).Filename, Start: start, End: end, Replacement: indentEncapsulateBlock(source, start, formatted.String())}, nil
}

func encapsulateAssignmentWritesField(pkg *packages.Package, assignment *ast.AssignStmt, fieldPosition token.Position) bool {
	if len(assignment.Lhs) != 1 {
		return false
	}
	selector, ok := assignment.Lhs[0].(*ast.SelectorExpr)
	if !ok {
		return false
	}
	selection := pkg.TypesInfo.Selections[selector]
	return selection != nil && sameGoObjectPosition(pkg.Fset, selection.Obj(), fieldPosition)
}

func encapsulateStatementListPosition(parent map[ast.Node]ast.Node, statement ast.Stmt) bool {
	switch owner := parent[statement].(type) {
	case *ast.BlockStmt:
		return slices.Contains(owner.List, statement)
	case *ast.CaseClause:
		return slices.Contains(owner.Body, statement)
	case *ast.CommClause:
		return slices.Contains(owner.Body, statement)
	default:
		return false
	}
}

func indentEncapsulateBlock(source []byte, start int, block string) string {
	lineStart := bytes.LastIndexByte(source[:start], '\n') + 1
	prefix := source[lineStart:start]
	if len(bytes.Trim(prefix, " \t")) != 0 {
		prefix = nil
	}
	return strings.ReplaceAll(block, "\n", "\n"+string(prefix))
}

func encapsulateSetterAssignment(receiverArgument ast.Expr, setter string, value ast.Expr) *ast.BlockStmt {
	receiver := ast.NewIdent(freshEncapsulateReceiverName(value))
	capture := &ast.AssignStmt{Lhs: []ast.Expr{receiver}, Tok: token.DEFINE, Rhs: []ast.Expr{receiverArgument}}
	call := &ast.CallExpr{Fun: &ast.SelectorExpr{X: receiver, Sel: ast.NewIdent(setter)}, Args: []ast.Expr{value}}
	return &ast.BlockStmt{List: []ast.Stmt{capture, &ast.ExprStmt{X: call}}}
}

func freshEncapsulateReceiverName(expression ast.Expr) string {
	used := make(map[string]struct{})
	ast.Inspect(expression, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if ok {
			used[identifier.Name] = struct{}{}
		}
		return true
	})
	for suffix := 0; ; suffix++ {
		candidate := "receiver"
		if suffix > 0 {
			candidate = fmt.Sprintf("receiver%d", suffix)
		}
		if _, exists := used[candidate]; !exists {
			return candidate
		}
	}
}

func encapsulateSetterReceiver(pkg *packages.Package, receiver *ast.Ident, ownerPosition token.Position) (ast.Expr, error) {
	typeOfReceiver := pkg.TypesInfo.TypeOf(receiver)
	if typeOfReceiver == nil {
		return nil, fmt.Errorf("encapsulate field cannot resolve setter receiver type")
	}
	switch typed := types.Unalias(typeOfReceiver).(type) {
	case *types.Pointer:
		named := encapsulateNamed(typed.Elem())
		if named == nil || !sameNamedGoObject(pkg.Fset, named, ownerPosition) {
			return nil, fmt.Errorf("encapsulate field refuses a pointer receiver with an unexpected type")
		}
		return ast.NewIdent(receiver.Name), nil
	case *types.Named:
		if !sameNamedGoObject(pkg.Fset, typed, ownerPosition) {
			return nil, fmt.Errorf("encapsulate field refuses a value receiver with an unexpected type")
		}
		return &ast.UnaryExpr{Op: token.AND, X: ast.NewIdent(receiver.Name)}, nil
	default:
		return nil, fmt.Errorf("encapsulate field requires a pointer or named value receiver")
	}
}

func selectorGetterRewrite(pkg *packages.Package, file *ast.File, selector *ast.SelectorExpr, getter string) (astedit.FieldReferenceRewrite, error) {
	start := pkg.Fset.PositionFor(selector.Pos(), false).Offset
	end := pkg.Fset.PositionFor(selector.End(), false).Offset
	if hasCommentInRange(pkg.Fset, file, start, end) {
		return astedit.FieldReferenceRewrite{}, fmt.Errorf("encapsulate field refuses comments inside rewritten field reads")
	}
	receiver := selector.X.(*ast.Ident)
	return astedit.FieldReferenceRewrite{File: pkg.Fset.PositionFor(selector.Pos(), false).Filename, Start: start, End: end, Replacement: receiver.Name + "." + getter + "()"}, nil
}

func encapsulateParents(file *ast.File) map[ast.Node]ast.Node {
	parents := make(map[ast.Node]ast.Node)
	stack := make([]ast.Node, 0, 32)
	ast.Inspect(file, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return false
		}
		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}
		stack = append(stack, node)
		return true
	})
	return parents
}

func sameGoObjectPosition(fset *token.FileSet, object types.Object, position token.Position) bool {
	if object == nil {
		return false
	}
	actual := fset.PositionFor(object.Pos(), false)
	actualPath, actualErr := filepath.Abs(actual.Filename)
	expectedPath, expectedErr := filepath.Abs(position.Filename)
	return actualErr == nil && expectedErr == nil && actualPath == expectedPath && actual.Offset == position.Offset
}

func keyedLiteralField(parent map[ast.Node]ast.Node, identifier *ast.Ident) bool {
	value, ok := parent[identifier].(*ast.KeyValueExpr)
	return ok && value.Key == identifier
}

func hasCommentInRange(fset *token.FileSet, file *ast.File, start, end int) bool {
	positionFile := fset.File(file.Pos())
	if positionFile == nil {
		return true
	}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if positionFile.Offset(comment.Pos()) < end && positionFile.Offset(comment.End()) > start {
				return true
			}
		}
	}
	return false
}

func resolveEncapsulatedMethod(ctx context.Context, root string, candidate SymbolTarget) (SymbolTarget, bool, error) {
	path := resolveWorkPath(root, candidate.File)
	source, err := sourcebuf.ReadFile(ctx, path)
	if err != nil {
		return SymbolTarget{}, false, fmt.Errorf("read accessor continuation %q: %w", path, err)
	}
	if candidate.ExpectedRevision == nil || structuralRevision(source) != *candidate.ExpectedRevision {
		return SymbolTarget{}, false, nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return SymbolTarget{}, false, fmt.Errorf("parse accessor continuation %q: %w", path, err)
	}
	if !sourceHasReceiverMethod(file, candidate.Symbol) {
		return SymbolTarget{}, false, nil
	}
	return candidate, true, nil
}

func sourceHasReceiverMethod(file *ast.File, symbol string) bool {
	parts := strings.Split(symbol, ").")
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "(*") {
		return false
	}
	owner, method := strings.TrimPrefix(parts[0], "(*"), parts[1]
	owner = strings.TrimSuffix(owner, " ")
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || function.Name.Name != method || len(function.Recv.List) != 1 {
			continue
		}
		star, ok := function.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		name, ok := star.X.(*ast.Ident)
		if ok && name.Name == owner {
			return true
		}
	}
	return false
}
