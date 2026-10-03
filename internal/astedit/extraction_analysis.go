// Package astedit provides bounded Go AST transformations.
// WHY: Binds compiler inputs to the source buffer before extraction publishes a candidate.
package astedit

import (
	"context"
	"fmt"
	"maps"

	"semedit/internal/gocache"
	"semedit/internal/sourcebuf"

	"golang.org/x/tools/go/packages"
)

func prepareExtractionBuffer(ctx context.Context, file string) (context.Context, *sourcebuf.Buffer, bool, error) {
	if sourcebuf.FromContext(ctx) != nil {
		return ctx, nil, false, nil
	}
	root, err := moveModuleRoot(file)
	if err != nil {
		return nil, nil, false, fmt.Errorf("locate extraction module: %w", err)
	}
	buffer, err := sourcebuf.New(root)
	if err != nil {
		return nil, nil, false, fmt.Errorf("capture extraction workspace: %w", err)
	}
	ctx = sourcebuf.WithBuffer(ctx, buffer)
	if err := buffer.AuthorizeWrites(ctx, root); err != nil {
		return nil, nil, false, fmt.Errorf("authorize extraction output: %w", err)
	}
	return ctx, buffer, true, nil
}

func loadExtractionPackages(ctx context.Context, dir string, overlay map[string][]byte, tests bool) ([]*packages.Package, error) {
	buffer := sourcebuf.FromContext(ctx)
	if buffer == nil {
		return nil, fmt.Errorf("extraction compiler analysis requires a captured source buffer")
	}
	bound := map[string]bool{"dir:" + dir: true}
	for {
		if err := sourcebuf.ObserveTree(ctx, dir); err != nil {
			return nil, fmt.Errorf("capture extraction package membership: %w", err)
		}
		current := sourcebuf.Overlay(ctx)
		if current == nil {
			current = make(map[string][]byte)
		}
		maps.Copy(current, overlay)
		environment, err := gocache.Environment(ctx, dir)
		if err != nil {
			return nil, fmt.Errorf("prepare extraction Go environment: %w", err)
		}
		config := &packages.Config{
			Context: ctx, Dir: dir, Env: environment, BuildFlags: []string{"-mod=readonly"}, Tests: tests, Overlay: current,
			Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax |
				packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps |
				packages.NeedModule | packages.NeedEmbedFiles,
		}
		loaded, err := packages.Load(config, ".")
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
