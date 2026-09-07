package processor

import (
	"encoding/json"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"testing"
)

func TestPDFIndirectActionPolicy(t *testing.T) {
	cases := []struct {
		name    string
		action  any
		objects map[string]any
		want    domain.Reason
	}{
		{"direct launch", "/Launch", nil, domain.UnsupportedContent},
		{"indirect launch", "6 0 R", map[string]any{"obj:6 0 R": map[string]any{"value": "/Launch"}}, domain.UnsupportedContent},
		{"chain launch", "6 0 R", map[string]any{"obj:6 0 R": map[string]any{"value": "7 0 R"}, "obj:7 0 R": map[string]any{"value": "/Launch"}}, domain.UnsupportedContent},
		{"cycle", "6 0 R", map[string]any{"obj:6 0 R": map[string]any{"value": "6 0 R"}}, domain.UnsupportedContent},
		{"missing", "6 0 R", nil, domain.UnsupportedContent},
		{"passive indirect", "6 0 R", map[string]any{"obj:6 0 R": map[string]any{"value": "/GoTo"}}, domain.NoReason},
		{"literal string", "u:6 0 R", nil, domain.NoReason},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			objects := map[string]any{"obj:1 0 R": map[string]any{"value": map[string]any{"/A": map[string]any{"/S": c.action}}}}
			for k, v := range c.objects {
				objects[k] = v
			}
			data, err := json.Marshal(map[string]any{"version": 2, "pages": []any{map[string]any{}}, "encrypt": map[string]any{"encrypted": false}, "qpdf": []any{map[string]any{}, objects}})
			if err != nil {
				t.Fatal(err)
			}
			var policy pdfStructure
			_, reason := policy.inspect(data)
			if reason != c.want {
				t.Fatalf("reason %s want %s", reason, c.want)
			}
		})
	}
}
