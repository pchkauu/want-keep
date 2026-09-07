package domain

import (
	"testing"
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func sampleAttachment() Attachment {
	return Attachment{Upload: Upload{ID: "00000000-0000-4000-8000-000000000001", AccountID: "00000000-0000-4000-8000-000000000002", Name: "receipt", MediaType: "image/png", Size: 1, Hash: Hash([]byte{1})}, ObjectID: "00000000-0000-4000-8000-000000000003", HouseholdID: "family", ActorID: "author", State: Uploaded, Reason: WaitingStorage, CreatedAt: time.Unix(1, 0)}
}
func TestAttachmentStatesAndImmutableIdentity(t *testing.T) {
	a := sampleAttachment()
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, modify := range []func(*Attachment){func(x *Attachment) { x.State = Accepted }, func(x *Attachment) { x.State = Validating }, func(x *Attachment) { x.Reason = InvalidDocument }, func(x *Attachment) { x.Upload.ID = "../receipt" }, func(x *Attachment) { x.Upload.MediaType = "image/svg+xml" }, func(x *Attachment) { x.Upload.Size = MaxBytes + 1 }, func(x *Attachment) { x.Upload.Name = "unsafe\r\nname" }} {
		b := a
		modify(&b)
		if b.Validate() == nil {
			t.Fatal("invalid state accepted")
		}
	}
	a.State = Validating
	a.Reason = NoReason
	a.OriginalReady = true
	issued := Attempt{Attachment: a, Token: "attempt", Number: 1, LeaseUntil: time.Now().Add(time.Minute)}
	if err := issued.RequireCurrent(issued, time.Now()); err != nil {
		t.Fatal(err)
	}
	for _, modify := range []func(*Attempt){func(x *Attempt) { x.Token = "other" }, func(x *Attempt) { x.Number++ }, func(x *Attempt) { x.Attachment.Upload.AccountID = "00000000-0000-4000-8000-000000000004" }, func(x *Attempt) { x.Attachment.HouseholdID = household.HouseholdID("other") }, func(x *Attempt) { x.Attachment.ActorID = "other" }, func(x *Attempt) { x.Attachment.State = Accepted }, func(x *Attempt) { x.LeaseUntil = time.Now().Add(-time.Second) }} {
		current := issued
		modify(&current)
		if issued.RequireCurrent(current, time.Now()) == nil {
			t.Fatal("stale result accepted")
		}
	}
}
