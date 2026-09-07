package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type contractBundle struct{ root string }

func (b contractBundle) read(_ *openapi3.Loader, uri *url.URL) ([]byte, error) {
	if uri.Scheme != "" || uri.Host != "" {
		return nil, errors.New("contract references must be local YAML")
	}
	resolved, err := filepath.EvalSymlinks(uri.Path)
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(b.root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.Ext(resolved) != ".yaml" {
		return nil, errors.New("contract reference outside API source")
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 1048576 {
		return nil, errors.New("invalid contract source file")
	}
	return os.ReadFile(resolved)
}

func (b contractBundle) build(source string) ([]byte, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = b.read
	document, err := loader.LoadFromFile(source)
	if err != nil {
		return nil, err
	}
	if err := document.Validate(context.Background()); err != nil {
		return nil, err
	}
	if err := b.internalize(document); err != nil {
		return nil, err
	}
	if err := document.Validate(context.Background()); err != nil {
		return nil, err
	}
	return json.MarshalIndent(document, "", "  ")
}

// Root component names remain stable when the same external schema is used by several paths.
func (b contractBundle) internalize(document *openapi3.T) error {
	names := make(map[string]string)
	register := func(name string, ref openapi3.ComponentRef) {
		if location := ref.RefPath(); location != nil {
			names[ref.CollectionName()+":"+location.String()] = name
		}
	}
	for name, ref := range document.Components.Schemas {
		register(name, ref)
	}
	for name, ref := range document.Components.Parameters {
		register(name, ref)
	}
	for name, ref := range document.Components.Responses {
		register(name, ref)
	}
	var resolutionError error
	document.InternalizeRefs(context.Background(), func(_ *openapi3.T, ref openapi3.ComponentRef) string {
		prefix := "#/components/" + ref.CollectionName() + "/"
		if strings.HasPrefix(ref.RefString(), prefix) {
			return strings.TrimPrefix(ref.RefString(), prefix)
		}
		if location := ref.RefPath(); location != nil {
			if name, ok := names[ref.CollectionName()+":"+location.String()]; ok {
				return name
			}
		}
		resolutionError = errors.New("external reference must be registered in root components")
		return "UnregisteredReference"
	})
	for _, ref := range document.Components.Responses {
		ref.Ref = ""
	}
	return resolutionError
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: contract-bundle <api/openapi.yaml>")
		os.Exit(2)
	}
	source, err := filepath.Abs(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	root, err := filepath.EvalSymlinks(filepath.Dir(source))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	data, err := (contractBundle{root: root}).build(source)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := os.Stdout.Write(data); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
