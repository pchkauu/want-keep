package processor

import (
	application "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
)

const ProtocolVersion = 1
const MaxResponseBytes = 64 * 1024 * 1024

type response struct {
	Version   int           `json:"version"`
	RequestID string        `json:"requestId"`
	Hash      string        `json:"hash"`
	Reason    domain.Reason `json:"reason"`
	Pages     []page        `json:"pages"`
}
type page struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	PNG    []byte `json:"png"`
}

func (r response) result() application.Inspection {
	x := application.Inspection{Reason: r.Reason, Pages: make([]application.Preview, len(r.Pages))}
	for i, p := range r.Pages {
		x.Pages[i] = application.Preview{Width: p.Width, Height: p.Height, PNG: p.PNG}
	}
	return x
}
