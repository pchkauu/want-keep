package architecture_test

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/pchkauu/want-keep/backend"

// Inner-layer third-party dependencies stay opt-in so framework and provider types cannot leak in unnoticed.
var approvedInnerThirdPartyPackages = []string{"github.com/cockroachdb/apd/v3"}

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
	writeGoFile(t, internalRoot, "accounts/domain/http_account.go", `package domain

import "net/http"

var _ = http.MethodGet
`)
	writeGoFile(t, internalRoot, "accounts/domain/persisted_account.go", `package domain

import _ "github.com/jackc/pgx/v5"
`)
	writeGoFile(t, internalRoot, "accounts/domain/coordinated_account.go", `package domain

import _ "github.com/pchkauu/want-keep/backend/internal/accounts/application"
`)
	writeGoFile(t, internalRoot, "accounts/domain/decimal_account.go", `package domain

import _ "github.com/cockroachdb/apd/v3"
`)
	writeGoFile(t, internalRoot, "accounts/application/stored_service.go", `package application

import _ "github.com/pchkauu/want-keep/backend/internal/storage"
`)
	writeGoFile(t, internalRoot, "accounts/application/http_service.go", `package application

import _ "net/http"
`)
	writeGoFile(t, internalRoot, "accounts/application/ai_service.go", `package application

import _ "github.com/pchkauu/want-keep/backend/internal/ai"
`)
	writeGoFile(t, internalRoot, "storage/repository.go", `package storage

import _ "github.com/pchkauu/want-keep/backend/internal/delivery"
`)

	violations, err := inspectImports(internalRoot)
	if err != nil {
		t.Fatalf("inspect fixture imports: %v", err)
	}
	got := make([]string, 0, len(violations))
	for _, violation := range violations {
		got = append(got, violation.String())
	}
	want := []string{
		`accounts/application/ai_service.go: application layer must not import "github.com/pchkauu/want-keep/backend/internal/ai"`,
		`accounts/application/http_service.go: application layer must not import "net/http"`,
		`accounts/application/stored_service.go: application layer must not import "github.com/pchkauu/want-keep/backend/internal/storage"`,
		`accounts/domain/coordinated_account.go: domain layer must not import "github.com/pchkauu/want-keep/backend/internal/accounts/application"`,
		`accounts/domain/decimal_account.go: domain layer must not import "github.com/cockroachdb/apd/v3"`,
		`accounts/domain/http_account.go: domain layer must not import "net/http"`,
		`accounts/domain/persisted_account.go: domain layer must not import "github.com/jackc/pgx/v5"`,
		`storage/repository.go: storage layer must not import "github.com/pchkauu/want-keep/backend/internal/delivery"`,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("violations = %q, want %q", got, want)
	}
}

func TestInspectImportsAllowsInwardDependencies(t *testing.T) {
	internalRoot := t.TempDir()
	writeGoFile(t, internalRoot, "money/domain/money.go", `package domain

import _ "github.com/cockroachdb/apd/v3"
`)
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
			if forbiddenImport(relative, layer, importPath) || (packageOrSubpackage(importPath, "github.com/cockroachdb/apd/v3") && !strings.HasPrefix(filepath.ToSlash(relative), "money/domain/")) {
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
	path := filepath.ToSlash(relative)
	if strings.HasPrefix(path, "connections/access/") {
		return "application"
	}
	for _, prefix := range []string{"attachments/files/", "attachments/processor/", "connections/credentials/", "privacy/cryptobox/"} {
		if strings.HasPrefix(path, prefix) {
			return "gateways"
		}
	}
	if strings.HasPrefix(filepath.ToSlash(relative), "identity/webauthn/") {
		return "gateways"
	}
	if strings.HasPrefix(filepath.ToSlash(relative), "connections/admission/") {
		return "application"
	}
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

func forbiddenImport(relative, layer, importPath string) bool {
	if layer == "application" && strings.HasPrefix(filepath.ToSlash(relative), "integrations/application/") && importPath == modulePath+"/internal/integrations/domain" {
		return false
	}
	for _, prefix := range []string{"attachments/files", "attachments/processor", "connections/credentials", "privacy/cryptobox"} {
		if (layer == "domain" || layer == "application" || layer == "ai") && packageOrSubpackage(importPath, modulePath+"/internal/"+prefix) {
			return true
		}
	}
	if layer == "domain" && packageOrSubpackage(importPath, modulePath+"/internal/connections/access") {
		return true
	}
	switch layer {
	case "domain":
		if strings.HasPrefix(importPath, modulePath+"/internal/connections/admission") {
			return true
		}
		return isForbiddenInnerImport(importPath, "application", "delivery", "storage", "integrations", "gateways", "ai")
	case "application":
		return isForbiddenInnerImport(importPath, "delivery", "storage", "integrations", "gateways", "ai")
	case "storage", "integrations", "gateways", "ai":
		return importsInternalLayer(importPath, "delivery")
	}
	return false
}

func isForbiddenInnerImport(importPath string, forbiddenLayers ...string) bool {
	if packageOrSubpackage(importPath, "net/http") || packageOrSubpackage(importPath, "database/sql") {
		return true
	}
	if importsInternalLayer(importPath, forbiddenLayers...) {
		return true
	}
	return isThirdPartyImport(importPath)
}

func importsInternalLayer(importPath string, layers ...string) bool {
	internalPath := strings.TrimPrefix(importPath, modulePath+"/internal/")
	if internalPath == importPath {
		return false
	}
	for _, part := range strings.Split(internalPath, "/") {
		if slices.Contains(layers, part) {
			return true
		}
	}
	return false
}

func isThirdPartyImport(importPath string) bool {
	if packageOrSubpackage(importPath, modulePath) {
		return false
	}
	firstPart, _, _ := strings.Cut(importPath, "/")
	if !strings.Contains(firstPart, ".") {
		return false
	}
	for _, approvedPackage := range approvedInnerThirdPartyPackages {
		if packageOrSubpackage(importPath, approvedPackage) {
			return false
		}
	}
	return true
}

func packageOrSubpackage(importPath, packagePath string) bool {
	return importPath == packagePath || strings.HasPrefix(importPath, packagePath+"/")
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

func TestAdmissionBoundaryIsApplicationOwned(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, root, "connections/admission/service.go", `package admission
import _ "github.com/pchkauu/want-keep/backend/internal/storage"
`)
	writeGoFile(t, root, "ledger/domain/admission.go", `package domain
import _ "github.com/pchkauu/want-keep/backend/internal/connections/admission"
`)
	violations, err := inspectImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 2 {
		t.Fatalf("admission dependency violations: %v", violations)
	}
}

func TestIngestionApplicationOwnsDomainWithoutTransportLeakage(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, root, "integrations/application/service.go", `package application
import _ "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
`)
	writeGoFile(t, root, "integrations/domain/transport.go", `package domain
import _ "github.com/pchkauu/want-keep/backend/internal/integrations/contract/generated"
`)
	writeGoFile(t, root, "accounts/application/ingestion.go", `package application
import _ "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
`)
	violations, err := inspectImports(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 2 || violations[0].importPath != modulePath+"/internal/integrations/domain" || violations[1].importPath != modulePath+"/internal/integrations/contract/generated" {
		t.Fatalf("ingestion dependency violations: %v", violations)
	}
}
