// Package astedit provides bounded Go AST transformations.
// WHY: Binds compiler inputs to the source buffer before extraction publishes a candidate.
package astedit

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"maps"
	"os"
	"path/filepath"

	"github.com/spockz/semantic-editor/internal/gocache"
	"github.com/spockz/semantic-editor/internal/sourcebuf"
	"github.com/spockz/semantic-editor/internal/telemetry"

	"golang.org/x/tools/go/packages"
)

type extractionLoadOptions struct {
	tests    bool
	typeInfo bool
}

func prepareExtractionBuffer(ctx context.Context, file string) (context.Context, *sourcebuf.Buffer, bool, error) {
	root, err := sourceBufferRoot(file)
	if err != nil {
		return nil, nil, false, fmt.Errorf("locate extraction module: %w", err)
	}
	preparedCtx, buffer, owned, err := sourcebuf.Prepare(ctx, root)
	if err != nil {
		return nil, nil, false, fmt.Errorf("prepare extraction buffer: %w", err)
	}
	if !owned {
		return preparedCtx, nil, false, nil
	}
	return preparedCtx, buffer, true, nil
}

func loadExtractionPackages(ctx context.Context, dir string, overlay map[string][]byte, options extractionLoadOptions) ([]*packages.Package, error) {
	buffer := sourcebuf.FromContext(ctx)
	if buffer == nil {
		return nil, fmt.Errorf("extraction compiler analysis requires a captured source buffer")
	}
	bound := map[string]bool{"dir:" + dir: true}
	for {
		if err := sourcebuf.ObserveTree(ctx, dir, ".go"); err != nil {
			return nil, fmt.Errorf("capture extraction package membership: %w", err)
		}
		current := sourcebuf.Overlay(ctx, ".go")
		if current == nil {
			current = make(map[string][]byte)
		}
		maps.Copy(current, overlay)
		environment, err := gocache.Environment(ctx, dir)
		if err != nil {
			return nil, fmt.Errorf("prepare extraction Go environment: %w", err)
		}
		buildFlags := []string{"-mod=readonly"}
		if _, found, moduleErr := findGoModuleRoot(filepath.Join(dir, "semedit-module-probe.go"), "extraction module"); moduleErr != nil {
			return nil, fmt.Errorf("locate extraction module: %w", moduleErr)
		} else if !found {
			buildFlags = nil
			environment = append(environment, "GO111MODULE=off")
		}
		mode := packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedTypes | packages.NeedImports | packages.NeedDeps |
			packages.NeedModule | packages.NeedEmbedFiles
		if options.typeInfo {
			mode |= packages.NeedSyntax | packages.NeedTypesInfo
		}
		config := &packages.Config{
			Context: ctx, Dir: dir, Env: environment, BuildFlags: buildFlags, Tests: options.tests, Overlay: current,
			Mode: mode,
		}
		stopLoad := telemetry.StartPhase(ctx, "go.extract.packages_load")
		loaded, err := packages.Load(config, ".")
		stopLoad()
		if err != nil {
			return nil, fmt.Errorf("load Go package for extraction: %w", err)
		}
		if len(loaded) == 0 {
			return nil, fmt.Errorf("load Go package for extraction: no package found")
		}
		for _, pkg := range loaded {
			for _, item := range pkg.Errors {
				return nil, fmt.Errorf("load Go package for extraction: %s", item.Error())
			}
		}
		added, err := bindExtractionPackageInputs(ctx, loaded, bound)
		if err != nil {
			return nil, fmt.Errorf("capture extraction compiler inputs: %w", err)
		}
		if added {
			continue
		}
		if err := buffer.AssertDisk(); err != nil {
			return nil, fmt.Errorf("revalidate extraction compiler inputs: %w", err)
		}
		return loaded, nil
	}
}

func typedExtractionFile(loaded []*packages.Package, filePath string) (*packages.Package, *token.FileSet, *ast.File, *types.Info, error) {
	absolute, err := filepath.Abs(filePath)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("resolve typed extraction file %q: %w", filePath, err)
	}
	for _, pkg := range loaded {
		if pkg.Fset == nil || pkg.TypesInfo == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			path, err := filepath.Abs(pkg.Fset.Position(file.Pos()).Filename)
			if err != nil {
				return nil, nil, nil, nil, fmt.Errorf("resolve typed extraction syntax path: %w", err)
			}
			if filepath.Clean(path) == filepath.Clean(absolute) {
				return pkg, pkg.Fset, file, pkg.TypesInfo, nil
			}
		}
	}
	return nil, nil, nil, nil, fmt.Errorf("typed extraction package did not include %s", filePath)
}

func runBufferedFile(ctx context.Context, file string, edit func(context.Context) error) error {
	root, err := sourceBufferRoot(file)
	if err != nil {
		return err
	}
	return sourcebuf.Run(ctx, root, edit)
}

func sourceBufferRoot(file string) (string, error) {
	absolute, err := filepath.Abs(file)
	if err != nil {
		return "", fmt.Errorf("resolve mutation file %q: %w", file, err)
	}
	if root, found, err := findGoModuleRoot(absolute, "mutation module"); err != nil {
		return "", err
	} else if found {
		return root, nil
	}
	return filepath.Dir(absolute), nil
}

func findGoModuleRoot(absPath, errContext string) (string, bool, error) {
	for dir := filepath.Dir(absPath); ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true, nil
		} else if !os.IsNotExist(err) {
			return "", false, fmt.Errorf("inspect %s at %s: %w", errContext, dir, err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, nil
		}
	}
}
