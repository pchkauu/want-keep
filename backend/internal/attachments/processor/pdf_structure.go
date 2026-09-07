package processor

import (
	"encoding/json"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"strings"
)

type pdfStructure struct {
	Version int               `json:"version"`
	Pages   []json.RawMessage `json:"pages"`
	Encrypt struct {
		Encrypted bool `json:"encrypted"`
	} `json:"encrypt"`
	Objects []json.RawMessage `json:"qpdf"`
	objects map[string]any
}

func (structure *pdfStructure) inspect(data []byte) (int, domain.Reason) {
	if json.Unmarshal(data, structure) != nil || structure.Version != 2 || len(structure.Objects) != 2 {
		return 0, domain.InvalidDocument
	}
	if structure.Encrypt.Encrypted {
		return 0, domain.UnsupportedContent
	}
	if len(structure.Pages) < 1 {
		return 0, domain.InvalidDocument
	}
	if len(structure.Pages) > domain.MaxPages {
		return 0, domain.LimitExceeded
	}
	if json.Unmarshal(structure.Objects[1], &structure.objects) != nil || len(structure.objects) == 0 {
		return 0, domain.InvalidDocument
	}
	remaining := 200000
	if !structure.passiveObject(structure.objects, 0, &remaining) {
		return 0, domain.UnsupportedContent
	}
	return len(structure.Pages), domain.NoReason
}
func (structure *pdfStructure) passiveObject(value any, depth int, remaining *int) bool {
	*remaining--
	if depth > 128 || *remaining < 0 {
		return false
	}
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			switch key {
			case "/JS", "/JavaScript", "/EmbeddedFiles", "/EF", "/RichMedia", "/RichMediaContent", "/XFA", "/OpenAction", "/AA", "/Movie", "/Sound", "/3D", "/3DD", "/RichMediaSettings":
				return false
			}
			if key == "/S" {
				resolved, ok := structure.resolve(item, remaining)
				if !ok {
					return false
				}
				if name, isName := resolved.(string); isName {
					switch name {
					case "/JavaScript", "/Launch", "/SubmitForm", "/ImportData", "/GoToR", "/GoToE", "/Rendition", "/Movie", "/Sound":
						return false
					}
				}
			}
			if !structure.passiveObject(item, depth+1, remaining) {
				return false
			}
		}
	case []any:
		for _, item := range v {
			if !structure.passiveObject(item, depth+1, remaining) {
				return false
			}
		}
	}
	return true
}

// qpdf represents an indirect object as "number generation R". Resolve only
// value chains here; walking every page/parent reference would follow normal PDF cycles.
func (structure *pdfStructure) resolve(value any, remaining *int) (any, bool) {
	seen := make(map[string]bool)
	for depth := 0; depth < 128; depth++ {
		reference, ok := value.(string)
		if !ok || strings.HasPrefix(reference, "/") || strings.HasPrefix(reference, "u:") || strings.HasPrefix(reference, "b:") || !strings.HasSuffix(reference, " R") {
			return value, true
		}
		*remaining--
		if *remaining < 0 || seen[reference] {
			return nil, false
		}
		seen[reference] = true
		object, ok := structure.objects["obj:"+reference].(map[string]any)
		if !ok {
			return nil, false
		}
		if next, exists := object["value"]; exists {
			value = next
			continue
		}
		if stream, exists := object["stream"]; exists {
			return stream, true
		}
		return nil, false
	}
	return nil, false
}
