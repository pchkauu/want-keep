package storage

import (
	"bytes"
	"encoding/json"
	"io"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
)

// bindingRow is the versioned persistence representation, independent of public API DTOs.
type bindingRow struct {
	Provider      string `json:"provider"`
	Environment   string `json:"environment"`
	Adapter       string `json:"adapter_build_digest"`
	Collector     string `json:"collector_image_digest"`
	Contract      string `json:"contract_version"`
	Allowlist     string `json:"allowlist_revision"`
	Configuration string `json:"non_secret_config_revision"`
	Permission    string `json:"operator_permission_revision"`
}

func bindingFromDomain(b connections.Binding) bindingRow {
	return bindingRow{b.Provider, b.Environment, b.AdapterBuildDigest, b.CollectorImageDigest, b.ContractVersion, b.AllowlistRevision, b.NonSecretConfigRevision, b.OperatorPermissionRevision}
}
func (r bindingRow) domain() (connections.Binding, error) {
	b := connections.Binding{Provider: r.Provider, Environment: r.Environment, AdapterBuildDigest: r.Adapter, CollectorImageDigest: r.Collector, ContractVersion: r.Contract, AllowlistRevision: r.Allowlist, NonSecretConfigRevision: r.Configuration, OperatorPermissionRevision: r.Permission}
	return b, b.Validate()
}
func decodeBinding(data []byte) (connections.Binding, error) {
	if len(data) > 4096 {
		return connections.Binding{}, connections.ErrInvalidAdmission
	}
	var row bindingRow
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&row); err != nil {
		return connections.Binding{}, connections.ErrInvalidAdmission
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return connections.Binding{}, connections.ErrInvalidAdmission
	}
	return row.domain()
}
