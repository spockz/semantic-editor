// Package cli builds registry-derived Cobra commands with ingress-specific presentation.
package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/token"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"semedit/internal/astedit"
	"semedit/internal/backend"
	"semedit/internal/maven"
	"semedit/internal/operation"
	"semedit/internal/pipeline"
	"semedit/internal/symbol"
)

// ErrCommandFailed marks expected command failures without printing Cobra usage.
var ErrCommandFailed = errors.New("command execution failed")

// FormatCLIError reports an operation failure, prefixing compiler positions when available.
func FormatCLIError(op string, err error) {
	var pos token.Position
	var synErr *astedit.SyntaxError
	var symErr *symbol.SymbolError
	var visErr *astedit.VisibilityMismatchError
	var placeErr *astedit.PlacementError

	switch {
	case errors.As(err, &synErr) && synErr.Pos.IsValid():
		pos = synErr.Pos
	case errors.As(err, &symErr) && symErr.Pos.IsValid():
		pos = symErr.Pos
	case errors.As(err, &visErr) && visErr.Pos.IsValid():
		pos = visErr.Pos
	case errors.As(err, &placeErr) && placeErr.Pos.IsValid():
		pos = placeErr.Pos
	}

	if pos.IsValid() {
		msg := err.Error()
		prefix := pos.String() + ": "
		if after, ok := strings.CutPrefix(msg, prefix); ok {
			msg = after
		}
		fmt.Fprintf(os.Stderr, "%s: %s\n", pos.String(), msg)
	} else {
		fmt.Fprintf(os.Stderr, "%s error: %v\n", op, err)
	}
}

// PrintDelta reports compiler diagnostic shifts: introductions to stderr,
// suggestions and resolutions to stdout.
func PrintDelta(delta pipeline.DiagnosticDelta) {
	if len(delta.Introduced) > 0 {
		fmt.Fprintf(os.Stderr, "diagnostics introduced:\n%s\n", strings.Join(delta.Introduced, "\n"))
	}
	if len(delta.Suggestions) > 0 {
		fmt.Printf("Actionable suggestions:\n- %s\n", strings.Join(delta.Suggestions, "\n- "))
	}
	if len(delta.Resolved) > 0 {
		fmt.Printf("diagnostics resolved:\n%s\n", strings.Join(delta.Resolved, "\n"))
	}
}

// flagValues holds one variable per registered flag, keyed by CLI flag name.
type flagValues struct {
	strings map[string]*string
	bools   map[string]*bool
	slices  map[string]*[]string
}

func stringDefault(value any) string {
	text, _ := value.(string)
	return text
}

func boolDefault(value any) bool {
	enabled, _ := value.(bool)
	return enabled
}

// registerFlags creates one long flag per parameter contract entry.
func registerFlags(cmd *cobra.Command, params []operation.ParameterContract, values *flagValues) {
	values.strings = map[string]*string{}
	values.bools = map[string]*bool{}
	values.slices = map[string]*[]string{}
	for _, param := range params {
		if param.CLIName == "" {
			continue
		}
		switch param.Type {
		case operation.ParamBoolean:
			values.bools[param.CLIName] = cmd.Flags().Bool(param.CLIName, boolDefault(param.Default), param.Description)
		case operation.ParamStringSlice:
			values.slices[param.CLIName] = cmd.Flags().StringArray(param.CLIName, nil, param.Description)
		default:
			values.strings[param.CLIName] = cmd.Flags().String(param.CLIName, stringDefault(param.Default), param.Description)
		}
	}
}

// rawParams translates flag values to the JSONName-keyed raw map consumed by Parse.
func rawParams(entry operation.Entry, values *flagValues) map[string]any {
	raw := make(map[string]any, len(entry.Params))
	for _, param := range entry.Params {
		switch param.Type {
		case operation.ParamBoolean:
			if variable, ok := values.bools[param.CLIName]; ok {
				raw[param.JSONName] = *variable
			}
		case operation.ParamStringSlice:
			if variable, ok := values.slices[param.CLIName]; ok {
				raw[param.JSONName] = append([]string(nil), *variable...)
			}
		default:
			if variable, ok := values.strings[param.CLIName]; ok {
				raw[param.JSONName] = *variable
			}
		}
	}
	return raw
}

// commandSpec adapts one registry operation to its historical CLI surface:
// validation messages, input normalization, output rendering, and error mapping.
type commandSpec struct {
	use       string
	short     string
	validate  func(values *flagValues) error
	adapt     func(cmd *cobra.Command, values *flagValues, raw map[string]any) error
	output    func(entry operation.Entry, values *flagValues, result any) error
	handleErr func(values *flagValues, err error) error
}

func requireFlags(message string, values *flagValues, names ...string) error {
	for _, name := range names {
		variable, ok := values.strings[name]
		if !ok || strings.TrimSpace(*variable) == "" {
			fmt.Fprintf(os.Stderr, "%s\n", message)
			return ErrCommandFailed
		}
	}
	return nil
}

func trimQuotes(text string) string {
	return strings.Trim(strings.TrimSpace(text), `"'`)
}

func stdoutJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "json format error: %v\n", err)
		return ErrCommandFailed
	}
	fmt.Println(string(data))
	return nil
}

func handleMavenError(op string, err error) error {
	var mavenErr *maven.Error
	if errors.As(err, &mavenErr) && mavenErr.MavenResult() != nil {
		payload := struct {
			Error  string        `json:"error"`
			Result *maven.Result `json:"result"`
		}{Error: err.Error(), Result: mavenErr.MavenResult()}
		if formatted, formatErr := json.MarshalIndent(payload, "", "  "); formatErr == nil {
			fmt.Fprintln(os.Stderr, string(formatted))
			return ErrCommandFailed
		}
	}
	FormatCLIError(op, err)
	return ErrCommandFailed
}

// fileEdit renders FileEditRes with a caller-supplied detail template.
func fileEdit(detail func(res operation.FileEditRes) string, withDiff bool) func(operation.Entry, *flagValues, any) error {
	return func(_ operation.Entry, _ *flagValues, result any) error {
		res, ok := result.(operation.FileEditRes)
		if !ok {
			return fmt.Errorf("unexpected result type %T: %w", result, ErrCommandFailed)
		}
		fmt.Println(detail(res))
		if withDiff && res.Diff != "" {
			fmt.Print(res.Diff)
		}
		PrintDelta(res.Delta)
		return nil
	}
}

func stdinFallback(cmd *cobra.Command, flagName, message string, values *flagValues) error {
	variable, ok := values.strings[flagName]
	if !ok {
		return fmt.Errorf("unknown flag %q: %w", flagName, ErrCommandFailed)
	}
	if strings.TrimSpace(*variable) != "" {
		return nil
	}
	data, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		fmt.Fprintf(os.Stderr, "read stdin: %v\n", err)
		return ErrCommandFailed
	}
	*variable = string(data)
	if strings.TrimSpace(*variable) == "" {
		fmt.Fprintf(os.Stderr, "%s\n", message)
		return ErrCommandFailed
	}
	return nil
}

// specs maps registry keys to their CLI surface. Verbs, messages, and outputs
// reproduce the historical hand-written commands byte for byte.
func specs() map[string]commandSpec {
	return map[string]commandSpec{
		"lookup": {
			use:   "lookup",
			short: "Resolve symbol location and AST coordinates",
			validate: func(values *flagValues) error {
				return requireFlags("lookup requires --symbol", values, "symbol")
			},
			output: func(_ operation.Entry, _ *flagValues, result any) error {
				return stdoutJSON(result)
			},
			handleErr: func(values *flagValues, err error) error {
				if errors.Is(err, symbol.ErrNotFound) {
					fmt.Fprintf(os.Stderr, "symbol not found: %s\n", strings.TrimSpace(*values.strings["symbol"]))
					return ErrCommandFailed
				}
				FormatCLIError("lookup", err)
				return ErrCommandFailed
			},
		},
		"rename": {
			use:   "rename",
			short: "Execute compiler-accurate semantic symbol rename across workspace",
			validate: func(values *flagValues) error {
				sym, to := values.strings["symbol"], values.strings["to"]
				if strings.TrimSpace(*sym) == "" || strings.TrimSpace(*to) == "" {
					fmt.Fprintf(os.Stderr, "rename requires --symbol and --to\n")
					return ErrCommandFailed
				}
				return nil
			},
			adapt: func(_ *cobra.Command, values *flagValues, raw map[string]any) error {
				*values.strings["symbol"] = trimQuotes(*values.strings["symbol"])
				*values.strings["to"] = trimQuotes(*values.strings["to"])
				raw["symbol"] = *values.strings["symbol"]
				raw["to"] = *values.strings["to"]
				raw["auto_organize_imports"] = false
				return nil
			},
			output: func(_ operation.Entry, _ *flagValues, result any) error {
				res, ok := result.(*backend.RenameResult)
				if !ok {
					return fmt.Errorf("unexpected result type %T: %w", result, ErrCommandFailed)
				}
				delta := res.Diagnostics
				if len(delta.Introduced) > 0 {
					fmt.Fprintf(os.Stderr, "diagnostics introduced:\n%s\n", strings.Join(delta.Introduced, "\n"))
				}
				if len(delta.Resolved) > 0 {
					fmt.Printf("diagnostics resolved:\n%s\n", strings.Join(delta.Resolved, "\n"))
				}
				return nil
			},
			handleErr: func(values *flagValues, err error) error {
				sym := strings.TrimSpace(*values.strings["symbol"])
				switch {
				case errors.Is(err, symbol.ErrNotFound):
					fmt.Fprintf(os.Stderr, "symbol not found: %s\n", sym)
				case errors.Is(err, backend.ErrAmbiguous):
					fmt.Fprintf(os.Stderr, "ambiguous symbol %q, please qualify receiver\n", sym)
				default:
					if _, ok := errors.AsType[*symbol.SymbolError](err); ok {
						FormatCLIError("rename resolution", err)
					} else {
						FormatCLIError("rename execution", err)
					}
				}
				return ErrCommandFailed
			},
		},
		"insert_declaration": {
			use:   "insert",
			short: "Insert top-level Go declaration into file",
			validate: func(values *flagValues) error {
				return requireFlags("insert requires --file and --source", values, "file", "source")
			},
			adapt: func(_ *cobra.Command, values *flagValues, raw map[string]any) error {
				trimmed := strings.TrimSpace(*values.strings["source"])
				if (strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"")) ||
					(strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'")) {
					trimmed = trimmed[1 : len(trimmed)-1]
				}
				*values.strings["source"] = trimmed
				*values.strings["target"] = trimQuotes(*values.strings["target"])
				raw["source"] = *values.strings["source"]
				raw["target_symbol"] = *values.strings["target"]
				return nil
			},
			output: fileEdit(func(res operation.FileEditRes) string {
				return fmt.Sprintf("Successfully inserted declaration into %s", res.Display)
			}, false),
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("insert", err)
				return ErrCommandFailed
			},
		},
		"insert_function": {
			use:   "insert-func",
			short: "Insert function or method with access modifier and section placement",
			validate: func(values *flagValues) error {
				return requireFlags("insert-func requires --file and --source", values, "file", "source")
			},
			output: fileEdit(func(res operation.FileEditRes) string {
				return fmt.Sprintf("Successfully inserted function into %s", res.Display)
			}, false),
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("insert-func", err)
				return ErrCommandFailed
			},
		},
		"insert_type": {
			use:   "insert-type",
			short: "Insert type declaration with access modifier and section placement",
			validate: func(values *flagValues) error {
				return requireFlags("insert-type requires --file and --source", values, "file", "source")
			},
			output: fileEdit(func(res operation.FileEditRes) string {
				return fmt.Sprintf("Successfully inserted type into %s", res.Display)
			}, false),
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("insert-type", err)
				return ErrCommandFailed
			},
		},
		"insert_decl": {
			use:   "insert-decl",
			short: "Insert general declaration (const, var) with grouping support",
			validate: func(values *flagValues) error {
				return requireFlags("insert-decl requires --file and --source", values, "file", "source")
			},
			output: fileEdit(func(res operation.FileEditRes) string {
				return fmt.Sprintf("Successfully inserted declaration into %s", res.Display)
			}, false),
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("insert-decl", err)
				return ErrCommandFailed
			},
		},
		"organize_imports": {
			use:   "imports",
			short: "Organize, sort, and reconcile import declarations",
			output: func(_ operation.Entry, _ *flagValues, result any) error {
				res, ok := result.(operation.FileEditRes)
				if !ok {
					return fmt.Errorf("unexpected result type %T: %w", result, ErrCommandFailed)
				}
				fmt.Println("Successfully organized imports.")
				PrintDelta(res.Delta)
				return nil
			},
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("organize imports", err)
				return ErrCommandFailed
			},
		},
		"replace_body": {
			use:   "replace-body",
			short: "Replace the body of an existing Go function or method by name",
			validate: func(values *flagValues) error {
				return requireFlags("replace-body requires --file and --symbol", values, "file", "symbol")
			},
			adapt: func(cmd *cobra.Command, values *flagValues, raw map[string]any) error {
				if err := stdinFallback(cmd, "body", "", values); err != nil {
					return err
				}
				raw["body"] = *values.strings["body"]
				return nil
			},
			output: fileEdit(func(res operation.FileEditRes) string {
				return fmt.Sprintf("Successfully replaced body of %s in %s", res.Symbol, res.Display)
			}, true),
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("replace-body", err)
				return ErrCommandFailed
			},
		},
		"scaffold_file": {
			use:   "scaffold-file",
			short: "Scaffold a new Go source file with package declaration",
			validate: func(values *flagValues) error {
				return requireFlags("scaffold-file requires --file", values, "file")
			},
			output: func(_ operation.Entry, _ *flagValues, result any) error {
				res, ok := result.(operation.ScaffoldFileRes)
				if !ok {
					return fmt.Errorf("unexpected result type %T: %w", result, ErrCommandFailed)
				}
				fmt.Printf("Successfully scaffolded %s with package %s\n", res.Display, res.Package)
				return nil
			},
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("scaffold-file", err)
				return ErrCommandFailed
			},
		},
		"insert_case": {
			use:   "insert-case",
			short: "Insert a case clause into an existing switch statement",
			validate: func(values *flagValues) error {
				return requireFlags("insert-case requires --file and --func", values, "file", "func")
			},
			adapt: func(cmd *cobra.Command, values *flagValues, raw map[string]any) error {
				if err := stdinFallback(cmd, "case", "insert-case requires case source via --case or stdin", values); err != nil {
					return err
				}
				raw["case"] = *values.strings["case"]
				return nil
			},
			output: fileEdit(func(res operation.FileEditRes) string {
				return fmt.Sprintf("Successfully inserted case into %s in %s", res.Symbol, res.Display)
			}, true),
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("insert-case", err)
				return ErrCommandFailed
			},
		},
		"verify": {
			use:   "verify",
			short: "Format sources and report compiler diagnostics without rolling back",
			output: func(entry operation.Entry, _ *flagValues, result any) error {
				text, err := entry.Format(result)
				if err != nil {
					return err
				}
				fmt.Println(text)
				return nil
			},
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("verify", err)
				return ErrCommandFailed
			},
		},
		"maven_compile": {
			use: "maven-compile", short: "Run fixed Maven test-compile for a trusted Java root POM",
			output: func(entry operation.Entry, _ *flagValues, result any) error {
				text, err := entry.Format(result)
				if err == nil {
					fmt.Println(text)
				}
				return err
			},
			handleErr: func(_ *flagValues, err error) error { return handleMavenError("maven-compile", err) },
		},
		"maven_test": {
			use: "maven-test", short: "Run fixed Maven test for a trusted Java root POM",
			output: func(entry operation.Entry, _ *flagValues, result any) error {
				text, err := entry.Format(result)
				if err == nil {
					fmt.Println(text)
				}
				return err
			},
			handleErr: func(_ *flagValues, err error) error { return handleMavenError("maven-test", err) },
		},
		"add_build_dependency": {
			use:   "add-build-dependency <package>",
			short: "Add external module to the build and tidy go.mod",
			adapt: func(cmd *cobra.Command, values *flagValues, raw map[string]any) error {
				pkg := trimQuotes(strings.TrimSpace(*values.strings["package"]))
				if pkg == "" && len(cmd.Flags().Args()) > 0 {
					pkg = trimQuotes(cmd.Flags().Args()[0])
				}
				if pkg == "" {
					fmt.Fprintf(os.Stderr, "add-build-dependency requires package name (e.g. semedit add-build-dependency github.com/google/uuid)\n")
					return ErrCommandFailed
				}
				*values.strings["package"] = pkg
				raw["package"] = pkg
				return nil
			},
			output: func(_ operation.Entry, _ *flagValues, result any) error {
				res, ok := result.(operation.BuildDependencyRes)
				if !ok {
					return fmt.Errorf("unexpected result type %T: %w", result, ErrCommandFailed)
				}
				fmt.Printf("Successfully added dependency %s\n", res.Package)
				return nil
			},
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("add build dependency", err)
				return ErrCommandFailed
			},
		},
		"snapshot": {
			use:   "snapshot [paths...]",
			short: "Capture pre-edit state into the transactional snapshot journal",
			adapt: func(cmd *cobra.Command, _ *flagValues, raw map[string]any) error {
				if args := cmd.Flags().Args(); len(args) > 0 {
					paths, _ := raw["paths"].([]string)
					raw["paths"] = append(paths, args...)
				}
				return nil
			},
			output: func(entry operation.Entry, _ *flagValues, result any) error {
				text, err := entry.Format(result)
				if err == nil {
					fmt.Println(text)
				}
				return err
			},
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("snapshot", err)
				return ErrCommandFailed
			},
		},
		"undo": {
			use:   "undo [snapshot-id]",
			short: "Restore a snapshot with conflict checking",
			adapt: func(cmd *cobra.Command, values *flagValues, raw map[string]any) error {
				id := strings.TrimSpace(*values.strings["id"])
				if args := cmd.Flags().Args(); len(args) > 0 && args[0] != "" {
					id = args[0]
				}
				if id == "" {
					fmt.Fprintln(os.Stderr, "undo requires snapshot ID (as argument or --id)")
					return ErrCommandFailed
				}
				raw["snapshot_id"] = id
				return nil
			},
			output: func(entry operation.Entry, _ *flagValues, result any) error {
				text, err := entry.Format(result)
				if err == nil {
					fmt.Println(text)
				}
				return err
			},
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("undo", err)
				return ErrCommandFailed
			},
		},
		"assertion_mode": {
			use:   "assertion-mode",
			short: "Rewrite Go test assertion failure mode safely",
			validate: func(values *flagValues) error {
				return requireFlags("assertion-mode requires --file and --mode", values, "file", "mode")
			},
			output: func(entry operation.Entry, _ *flagValues, result any) error {
				text, err := entry.Format(result)
				if err == nil {
					fmt.Println(text)
				}
				return err
			},
			handleErr: func(_ *flagValues, err error) error {
				FormatCLIError("assertion-mode", err)
				return ErrCommandFailed
			},
		},
	}
}

// commandOrder fixes help listing and generation order.
var commandOrder = []string{
	"lookup", "rename", "verify", "maven_compile", "maven_test",
	"insert_declaration", "insert_function", "insert_type", "insert_decl",
	"organize_imports", "add_build_dependency",
	"replace_body", "scaffold_file", "insert_case",
	"snapshot", "undo",
	"assertion_mode",
}

// Commands builds one Cobra command per registry operation with a CLI name, bound to workDir.
func Commands(workDir string) []*cobra.Command {
	registry := operation.DefaultRegistry()
	defined := specs()
	commands := make([]*cobra.Command, 0, len(commandOrder))
	for _, key := range commandOrder {
		entry, ok := registry.LookupKey(key)
		if !ok {
			panic("cli command without registry operation: " + key)
		}
		spec, ok := defined[key]
		if !ok {
			panic("registry operation without CLI surface: " + key)
		}
		commands = append(commands, buildCommand(workDir, registry, entry, spec))
	}
	return commands
}

func buildCommand(workDir string, registry *operation.Registry, entry operation.Entry, spec commandSpec) *cobra.Command {
	values := &flagValues{}
	cmd := &cobra.Command{
		Use:           spec.use,
		Short:         spec.short,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if spec.validate != nil {
				if err := spec.validate(values); err != nil {
					return err
				}
			}
			raw := rawParams(entry, values)
			if spec.adapt != nil {
				if err := spec.adapt(cmd, values, raw); err != nil {
					return err
				}
			}
			cc := operation.NewCallContext(workDir, backend.ProjectContext{})
			cc.Ctx = cmd.Context()
			result, err := registry.Dispatch(cc, entry.Key, raw)
			if err != nil {
				if spec.handleErr != nil {
					return spec.handleErr(values, err)
				}
				FormatCLIError(entry.Key, err)
				return ErrCommandFailed
			}
			if spec.output != nil {
				return spec.output(entry, values, result)
			}
			text, err := entry.Format(result)
			if err != nil {
				return err
			}
			fmt.Println(text)
			return nil
		},
	}
	registerFlags(cmd, entry.Params, values)
	return cmd
}
