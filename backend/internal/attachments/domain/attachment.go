package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

const MaxBytes = 10 * 1024 * 1024
const MaxPages = 10
const MaxPreviewBytes = 17 * 1024 * 1024

var (
	ErrInvalid     = errors.New("invalid attachment")
	ErrNotFound    = errors.New("attachment not found")
	ErrConflict    = errors.New("upload content conflict")
	ErrNotReady    = errors.New("attachment not ready")
	ErrUnavailable = errors.New("private storage unavailable")
	ErrStale       = errors.New("stale attachment attempt")
)
var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type State string

const (
	Uploaded   State = "uploaded"
	Validating State = "validating"
	Accepted   State = "accepted"
	Rejected   State = "rejected"
)

type Reason string

const (
	NoReason           Reason = ""
	WaitingStorage     Reason = "storage_unavailable"
	WaitingProcessor   Reason = "processor_unavailable"
	InvalidDocument    Reason = "invalid_document"
	UnsupportedContent Reason = "unsupported_content"
	LimitExceeded      Reason = "limit_exceeded"
)

func (r Reason) Valid() bool {
	switch r {
	case NoReason, WaitingStorage, WaitingProcessor, InvalidDocument, UnsupportedContent, LimitExceeded:
		return true
	}
	return false
}
func (r Reason) Rejects() bool {
	return r == InvalidDocument || r == UnsupportedContent || r == LimitExceeded
}

type Upload struct {
	ID, AccountID, Name, MediaType string
	Size                           int64
	Hash                           string
}

func (u Upload) Validate() error {
	if !uuidPattern.MatchString(u.ID) || !uuidPattern.MatchString(u.AccountID) || len(u.Name) == 0 || utf8.RuneCountInString(u.Name) > 2000 || strings.ContainsAny(u.Name, "\x00\r\n") || !utf8.ValidString(u.Name) || u.Size < 1 || u.Size > MaxBytes || !digestPattern.MatchString(u.Hash) {
		return ErrInvalid
	}
	switch u.MediaType {
	case "image/jpeg", "image/png", "image/webp", "application/pdf":
		return nil
	}
	return ErrInvalid
}
func Hash(data []byte) string { x := sha256.Sum256(data); return hex.EncodeToString(x[:]) }

type Attachment struct {
	Upload        Upload
	HouseholdID   household.HouseholdID
	ActorID       household.UserID
	ObjectID      string
	State         State
	Reason        Reason
	OriginalReady bool
	Pages         []Page
	CreatedAt     time.Time
}
type Page struct {
	Number, Width, Height int
	Hash                  string
	Size                  int64
}

func (a Attachment) Validate() error {
	if a.Upload.Validate() != nil || a.HouseholdID == "" || a.ActorID == "" || !uuidPattern.MatchString(a.ObjectID) || a.CreatedAt.IsZero() || !a.Reason.Valid() {
		return ErrInvalid
	}
	switch a.State {
	case Uploaded, Validating, Accepted, Rejected:
	default:
		return ErrInvalid
	}
	if a.State != Uploaded && !a.OriginalReady {
		return ErrInvalid
	}
	if a.State == Validating && a.Reason != NoReason {
		return ErrInvalid
	}
	if a.State == Uploaded && a.Reason.Rejects() {
		return ErrInvalid
	}
	if a.State == Accepted && a.Upload.MediaType != "application/pdf" && len(a.Pages) != 1 {
		return ErrInvalid
	}
	if a.State != Accepted && len(a.Pages) != 0 {
		return ErrInvalid
	}
	if a.State == Accepted {
		if !a.OriginalReady || a.Reason != NoReason || len(a.Pages) < 1 || len(a.Pages) > MaxPages {
			return ErrInvalid
		}
		for i, p := range a.Pages {
			if p.Number != i+1 || p.Width < 1 || p.Height < 1 || p.Width > 2048 || p.Height > 2048 || p.Size < 1 || p.Size > MaxPreviewBytes || !digestPattern.MatchString(p.Hash) {
				return ErrInvalid
			}
		}
	}
	if a.State == Rejected && !a.Reason.Rejects() {
		return ErrInvalid
	}
	return nil
}
func (a Attachment) Replay(p household.Principal, u Upload) error {
	if p.HouseholdID() != a.HouseholdID || p.UserID() != a.ActorID || a.Upload != u {
		return ErrConflict
	}
	return nil
}
func (a Attachment) RequireRead(p household.Principal) error {
	if p.RequireHousehold(a.HouseholdID) != nil {
		return ErrNotFound
	}
	return a.Validate()
}
func (a Attachment) RequireContent(p household.Principal) error {
	if err := a.RequireRead(p); err != nil {
		return err
	}
	if a.State != Accepted {
		return ErrNotReady
	}
	return nil
}
func (a Attachment) Object(page int) Object {
	purpose := "original"
	size := a.Upload.Size
	hash := a.Upload.Hash
	if page > 0 {
		purpose = fmt.Sprintf("preview-%d", page)
		size = 0
		hash = ""
		if page <= len(a.Pages) {
			size = a.Pages[page-1].Size
			hash = a.Pages[page-1].Hash
		}
	}
	return Object{HouseholdID: a.HouseholdID, AttachmentID: a.Upload.ID, ObjectID: a.ObjectID, Purpose: purpose, Size: size, Hash: hash}
}

type Object struct {
	HouseholdID                     household.HouseholdID
	AttachmentID, ObjectID, Purpose string
	Size                            int64
	Hash                            string
}

func (o Object) Validate() error {
	if o.HouseholdID == "" || !uuidPattern.MatchString(o.AttachmentID) || !uuidPattern.MatchString(o.ObjectID) || o.Size < 1 || o.Size > MaxPreviewBytes || !digestPattern.MatchString(o.Hash) {
		return ErrInvalid
	}
	if o.Purpose == "original" {
		if o.Size > MaxBytes {
			return ErrInvalid
		}
		return nil
	}
	for i := 1; i <= MaxPages; i++ {
		if o.Purpose == fmt.Sprintf("preview-%d", i) {
			return nil
		}
	}
	return ErrInvalid
}

type Attempt struct {
	Attachment Attachment
	Token      string
	Number     int
	LeaseUntil time.Time
}

func (a Attempt) RequireCurrent(current Attempt, now time.Time) error {
	if a.Token == "" || a.Token != current.Token || a.Number != current.Number || a.Attachment.Upload != current.Attachment.Upload || a.Attachment.ObjectID != current.Attachment.ObjectID || a.Attachment.ActorID != current.Attachment.ActorID || a.Attachment.HouseholdID != current.Attachment.HouseholdID || !now.Before(current.LeaseUntil) || current.Attachment.State != Validating {
		return ErrStale
	}
	return nil
}
