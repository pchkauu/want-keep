package main

import (
	"context"
	"github.com/getkin/kin-openapi/openapi3"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestReferenceReaderRejectsNetworkAndEscapes(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "local.yaml")
	if err := os.WriteFile(source, []byte("openapi: 3.0.3"), 0600); err != nil {
		t.Fatal(err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	bundle := contractBundle{root: resolvedRoot}
	if _, err := bundle.read(nil, &url.URL{Path: source}); err != nil {
		t.Fatal(err)
	}
	for _, uri := range []*url.URL{{Scheme: "https", Host: "example.invalid", Path: "/schema.yaml"}, {Scheme: "file", Path: source}, {Path: filepath.Join(root, "missing.yaml")}} {
		if _, err := bundle.read(nil, uri); err == nil {
			t.Fatal("unsafe reference accepted")
		}
	}
	outside := t.TempDir()
	external := filepath.Join(outside, "outside.yaml")
	if err := os.WriteFile(external, []byte("private"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link.yaml")
	if err := os.Symlink(external, link); err != nil {
		t.Fatal(err)
	}
	if _, err := bundle.read(nil, &url.URL{Path: link}); err == nil {
		t.Fatal("symlink escaped source root")
	}
}

func TestBundlePreservesRootNamesAndReusableResponses(t *testing.T) {
	source, err := filepath.Abs("../../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(filepath.Dir(source))
	if err != nil {
		t.Fatal(err)
	}
	data, err := (contractBundle{root: root}).build(source)
	if err != nil {
		t.Fatal(err)
	}
	loader := openapi3.NewLoader()
	document, err := loader.LoadFromData(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if document.Components.Schemas["Money"] == nil || document.Components.Schemas["Money"].Value == nil {
		t.Fatal("Money name or schema lost")
	}
	for path, item := range document.Paths.Map() {
		for method, operation := range item.Operations() {
			for status, response := range operation.Responses.Map() {
				if response.Value == nil {
					t.Fatalf("unresolved response %s %s %s", method, path, status)
				}
			}
		}
	}
	if document.Components.Responses["Problem"].Ref != "" {
		t.Fatal("root response must have a value instead of a self reference")
	}
}
