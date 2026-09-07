package storage

import (
	"encoding/json"
	"strings"
	"testing"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
)

func TestPersistenceBindingIsClosedAndExact(t *testing.T) {
	b := connections.Binding{Provider: "emcd", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("a", 64), CollectorImageDigest: "sha256:" + strings.Repeat("b", 64), ContractVersion: "10", AllowlistRevision: "1", NonSecretConfigRevision: "1", OperatorPermissionRevision: "1"}
	raw, err := json.Marshal(bindingFromDomain(b))
	if err != nil {
		t.Fatal(err)
	}
	roundtrip, err := decodeBinding(raw)
	if err != nil || roundtrip != b {
		t.Fatal("binding lost fields")
	}
	for _, bad := range []string{`{}`, `null`, string(raw) + ` {}`, strings.Replace(string(raw), `"contract_version":"10"`, `"contract_version":10`, 1), strings.TrimSuffix(string(raw), "}") + `,"admitted":true}`} {
		if _, err = decodeBinding([]byte(bad)); err == nil {
			t.Fatal("invalid binding accepted")
		}
	}
}
