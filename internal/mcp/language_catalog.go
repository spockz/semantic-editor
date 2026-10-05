// Package mcp stores immutable language snapshots and tool catalogs for one MCP server.
package mcp

import (
	"errors"
	"reflect"
	"semedit/internal/backend"
	"sync"
)

var errCatalogGenerationChanged = errors.New("MCP catalog generation changed during build")

type catalogCache struct {
	mu         sync.Mutex
	generation uint64
	languages  map[string][]backend.LanguageID
	catalogs   map[catalogKey][]map[string]any
	flights    map[catalogKey]*catalogFlight
}

type catalogKey struct {
	root       string
	generation uint64
}

type catalogFlight struct {
	generation uint64
	done       chan struct{}
}

func newCatalogCache() *catalogCache {
	return &catalogCache{languages: make(map[string][]backend.LanguageID), catalogs: make(map[catalogKey][]map[string]any), flights: make(map[catalogKey]*catalogFlight)}
}

func (c *catalogCache) reset(root string, languages []backend.LanguageID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.generation++
	c.languages = make(map[string][]backend.LanguageID)
	c.catalogs = make(map[catalogKey][]map[string]any)
	c.flights = make(map[catalogKey]*catalogFlight)
	if root != "" {
		c.languages[backend.CanonicalWorkspaceRoot(root)] = append([]backend.LanguageID(nil), languages...)
	}
}

func (c *catalogCache) toolCatalogForGeneration(root string, generation uint64, build func() ([]map[string]any, error)) ([]map[string]any, error) {
	root = backend.CanonicalWorkspaceRoot(root)
	for {
		c.mu.Lock()
		if c.generation != generation {
			c.mu.Unlock()
			return nil, errCatalogGenerationChanged
		}
		key := catalogKey{root: root, generation: generation}
		if catalog, ok := c.catalogs[key]; ok {
			c.mu.Unlock()
			return catalog, nil
		}
		if flight, ok := c.flights[key]; ok {
			done := flight.done
			c.mu.Unlock()
			<-done
			continue
		}
		flight := &catalogFlight{generation: generation, done: make(chan struct{})}
		c.flights[key] = flight
		c.mu.Unlock()

		catalog, err := build()
		c.mu.Lock()
		delete(c.flights, key)
		stale := c.generation != generation
		if err == nil && !stale {
			c.catalogs[key] = catalog
		}
		close(flight.done)
		c.mu.Unlock()
		if stale {
			return nil, errCatalogGenerationChanged
		}
		return catalog, err
	}
}

func (c *catalogCache) languageSnapshot(root string, detect func() ([]backend.LanguageID, error)) ([]backend.LanguageID, error) {
	root = backend.CanonicalWorkspaceRoot(root)
	c.mu.Lock()
	defer c.mu.Unlock()
	if languages, ok := c.languages[root]; ok {
		return append([]backend.LanguageID(nil), languages...), nil
	}
	languages, err := detect()
	if err != nil {
		return nil, err
	}
	frozen := append([]backend.LanguageID(nil), languages...)
	c.languages[root] = frozen
	return append([]backend.LanguageID(nil), frozen...), nil
}

func cloneToolCatalog(tools []map[string]any) []map[string]any {
	cloned := make([]map[string]any, len(tools))
	for index, tool := range tools {
		clonedTool := make(map[string]any, len(tool))
		for key, value := range tool {
			clonedTool[key] = cloneCatalogValue(value)
		}
		cloned[index] = clonedTool
	}
	return cloned
}

func cloneCatalogValue(value any) any {
	if value == nil {
		return nil
	}
	return cloneCatalogReflect(reflect.ValueOf(value)).Interface()
}

func (c *catalogCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.generation++
	c.languages = make(map[string][]backend.LanguageID)
	c.catalogs = make(map[catalogKey][]map[string]any)
	c.flights = make(map[catalogKey]*catalogFlight)
}

func (c *catalogCache) currentGeneration() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.generation
}

func (c *catalogCache) toolCatalog(root string, build func() ([]map[string]any, error)) ([]map[string]any, error) {
	return c.toolCatalogForGeneration(root, c.currentGeneration(), build)
}

func cloneCatalogReflect(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := reflect.New(value.Type()).Elem()
		cloned.Set(cloneCatalogReflect(value.Elem()))
		return cloned
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := reflect.MakeMapWithSize(value.Type(), value.Len())
		iterator := value.MapRange()
		for iterator.Next() {
			cloned.SetMapIndex(cloneCatalogReflect(iterator.Key()), cloneCatalogReflect(iterator.Value()))
		}
		return cloned
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for index := range value.Len() {
			cloned.Index(index).Set(cloneCatalogReflect(value.Index(index)))
		}
		return cloned
	case reflect.Array:
		cloned := reflect.New(value.Type()).Elem()
		for index := range value.Len() {
			cloned.Index(index).Set(cloneCatalogReflect(value.Index(index)))
		}
		return cloned
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		cloned := reflect.New(value.Type().Elem())
		cloned.Elem().Set(cloneCatalogReflect(value.Elem()))
		return cloned
	case reflect.Struct:
		cloned := reflect.New(value.Type()).Elem()
		cloned.Set(value)
		for index := range value.NumField() {
			if value.Type().Field(index).IsExported() {
				cloned.Field(index).Set(cloneCatalogReflect(value.Field(index)))
			}
		}
		return cloned
	default:
		return value
	}
}
