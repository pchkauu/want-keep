package architecture_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/pchkauu/want-keep/backend"

type importViolation struct {
	file       string
	layer      string
	importPath string
}

func (violation importViolation) String() string {
	return fmt.Sprintf("%s: %s layer must not import %q", violation.file, violation.layer, violation.importPath)
}

func TestRepositoryLayerImports(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve architecture test location")
	}

	internalRoot := filepath.Dir(filepath.Dir(filename))
	violations, err := inspectImports(internalRoot)
	if err != nil {
		t.Fatalf("inspect imports: %v", err)
	}
	if len(violations) == 0 {
		return
	}

	for _, violation := range violations {
		t.Error(violation)
	}
}

func TestInspectImportsRejectsOutwardDependencies(t *testing.T) {
	internalRoot := t.TempDir()
	writeGoFile(t, internalRoot, "accounts/domain/account.go", `package domain

import "net/http"

var _ = http.MethodGet
`)
	writeGoFile(t, internalRoot, "accounts/application/service.go", `package application

import _ "github.com/pchkauu/want-keep/backend/internal/storage"
`)
	writeGoFile(t, internalRoot, "storage/repository.go", `package storage

import _ "github.com/pchkauu/want-keep/backend/internal/delivery"
`)

	violations, err := inspectImports(internalRoot)
	if err != nil {
		t.Fatalf("inspect fixture imports: %v", err)
	}
	if got, want := len(violations), 3; got != want {
		t.Fatalf("violation count = %d, want %d: %v", got, want, violations)
	}
}

func TestInspectImportsAllowsInwardDependencies(t *testing.T) {
	internalRoot := t.TempDir()
	writeGoFile(t, internalRoot, "accounts/domain/account.go", `package domain

import "time"

var _ = time.Time{}
`)
	writeGoFile(t, internalRoot, "accounts/application/service.go", `package application

import _ "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
`)
	writeGoFile(t, internalRoot, "storage/repository.go", `package storage

import _ "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
`)

	violations, err := inspectImports(internalRoot)
	if err != nil {
		t.Fatalf("inspect fixture imports: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("unexpected violations: %v", violations)
	}
}

func inspectImports(internalRoot string) ([]importViolation, error) {
	var violations []importViolation
	err := filepath.WalkDir(internalRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		relative, err := filepath.Rel(internalRoot, path)
		if err != nil {
			return err
		}
		layer := owningLayer(relative)
		if layer == "" {
			return nil
		}

		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse %s: %w", relative, err)
		}
		for _, imported := range parsed.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return fmt.Errorf("parse import in %s: %w", relative, err)
			}
			if forbiddenImport(layer, importPath) {
				violations = append(violations, importViolation{
					file:       filepath.ToSlash(relative),
					layer:      layer,
					importPath: importPath,
				})
			}
		}
		return nil
	})
	sort.Slice(violations, func(left, right int) bool {
		return violations[left].String() < violations[right].String()
	})
	return violations, err
}

func owningLayer(relative string) string {
	parts := strings.Split(filepath.ToSlash(relative), "/")
	for _, part := range parts[:len(parts)-1] {
		switch part {
		case "domain", "application":
			return part
		}
	}
	if len(parts) < 2 {
		return ""
	}
	switch parts[0] {
	case "delivery", "storage", "integrations", "gateways", "ai":
		return parts[0]
	default:
		return ""
	}
}

func forbiddenImport(layer, importPath string) bool {
	outerPackages := []string{"/delivery", "/storage", "/integrations", "/gateways"}
	switch layer {
	case "domain":
		if importPath == "net/http" || strings.HasPrefix(importPath, "net/http/") || importPath == "database/sql" || strings.HasPrefix(importPath, "database/sql/") || strings.Contains(importPath, "openai") || strings.Contains(importPath, "playwright") {
			return true
		}
		for _, outerPackage := range append(outerPackages, "/ai") {
			if strings.HasPrefix(importPath, modulePath+"/internal"+outerPackage) {
				return true
			}
		}
	case "application":
		for _, outerPackage := range outerPackages {
			if strings.HasPrefix(importPath, modulePath+"/internal"+outerPackage) {
				return true
			}
		}
	case "storage", "integrations", "gateways", "ai":
		return strings.HasPrefix(importPath, modulePath+"/internal/delivery")
	}
	return false
}

func writeGoFile(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
