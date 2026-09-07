//go:build integration

package privacy_test

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	domain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
)

type syntheticDocument struct {
	media string
	data  []byte
	pages int
}

func imageDocument(t *testing.T, media string) []byte {
	t.Helper()
	if media == "image/webp" {
		data, err := base64.StdEncoding.DecodeString("UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==")
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	canvas := image.NewNRGBA(image.Rect(0, 0, 19, 13))
	canvas.Set(0, 0, color.NRGBA{R: 77, A: 255})
	var b bytes.Buffer
	var err error
	if media == "image/jpeg" {
		err = jpeg.Encode(&b, canvas, nil)
	} else {
		err = png.Encode(&b, canvas)
	}
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func pdfDocument(pages int, extra string) []byte {
	objects := []string{"<< /Type /Catalog /Pages 2 0 R " + extra + " >>", ""}
	kids := []string{}
	for i := 0; i < pages; i++ {
		id := len(objects) + 1
		kids = append(kids, fmt.Sprintf("%d 0 R", id))
		objects = append(objects, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 120 80] /Resources << >> /Contents %d 0 R >>", id+1), "<< /Length 0 >>\nstream\n\nendstream")
	}
	objects[1] = fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>", pages, strings.Join(kids, " "))
	var b bytes.Buffer
	b.WriteString("%PDF-1.7\n")
	offsets := []int{0}
	for i, o := range objects {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	start := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets))
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), start)
	return b.Bytes()
}
func TestFourFormatsFamilyReadAndEveryPDFPage(t *testing.T) {
	f := newFixture(t)
	documents := []syntheticDocument{{"image/png", imageDocument(t, "image/png"), 1}, {"image/jpeg", imageDocument(t, "image/jpeg"), 1}, {"image/webp", imageDocument(t, "image/webp"), 1}, {"application/pdf", pdfDocument(10, ""), 10}}
	for _, doc := range documents {
		t.Run(doc.media, func(t *testing.T) {
			id := uuid.NewString()
			requireStatus(t, f.upload(f.ta, id, f.accountID, doc.media, doc.data), 200)
			requireStatus(t, f.request(f.tb, "GET", "/attachments/"+id+"/content", nil, ""), 409)
			if err := f.service.ProcessNext(testContext); err != nil {
				t.Fatal(err)
			}
			a := f.metadata(f.tb, id)
			if a.State != domain.Accepted || len(a.Pages) != doc.pages {
				t.Fatalf("not accepted: %s %s pages=%d", a.State, a.Reason, len(a.Pages))
			}
			original := f.request(f.tb, "GET", "/attachments/"+id+"/content", nil, "")
			requireStatus(t, original, 200)
			if !bytes.Equal(original.Body.Bytes(), doc.data) || original.Header().Get("Content-Type") != "application/octet-stream" || !strings.HasPrefix(original.Header().Get("Content-Disposition"), "attachment;") {
				t.Fatal("original download changed or unsafe")
			}
			for page := 1; page <= doc.pages; page++ {
				w := f.request(f.tb, "GET", fmt.Sprintf("/attachments/%s/preview/%d", id, page), nil, "")
				requireStatus(t, w, 200)
				if w.Header().Get("Content-Type") != "image/png" || w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Content-Security-Policy") == "" {
					t.Fatal("unsafe preview headers")
				}
				if _, err := png.Decode(bytes.NewReader(w.Body.Bytes())); err != nil {
					t.Fatal(err)
				}
			}
			requireStatus(t, f.request(f.tf, "GET", "/attachments/"+id, nil, ""), 404)
			requireStatus(t, f.request(f.tf, "GET", "/attachments/"+uuid.NewString(), nil, ""), 404)
			requireStatus(t, f.request("", "GET", "/attachments/"+id+"/content", nil, ""), 401)
			data, media, err := f.service.ReadForAnalysis(testContext, f.b, id)
			if err != nil || media != doc.media || !bytes.Equal(data, doc.data) {
				t.Fatal("accepted AI read failed")
			}
		})
	}
	var operations, commands int
	if err := f.admin.QueryRow(testContext, `SELECT (SELECT count(*) FROM want_keep.postings),(SELECT count(*) FROM want_keep.command_tombstones)`).Scan(&operations, &commands); err != nil || operations != 0 || commands != 0 {
		t.Fatal("upload created financial effects", err)
	}
}
func TestRejectInvalidAndActiveDocuments(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		name, media string
		data        []byte
		reason      domain.Reason
	}{
		{"mime", "image/jpeg", imageDocument(t, "image/png"), domain.UnsupportedContent},
		{"corrupt-image", "image/png", []byte("not a png"), domain.InvalidDocument},
		{"corrupt-pdf", "application/pdf", []byte("%PDF-1.7\nbroken"), domain.InvalidDocument},
		{"pages", "application/pdf", pdfDocument(11, ""), domain.LimitExceeded},
		{"javascript", "application/pdf", pdfDocument(1, "/OpenAction << /S /JavaScript /JS (synthetic) >>"), domain.UnsupportedContent},
		{"launch", "application/pdf", pdfDocument(1, "/Names << /Actions << /S /Launch /F (synthetic) >> >>"), domain.UnsupportedContent},
		{"embedded", "application/pdf", pdfDocument(1, "/Names << /EmbeddedFiles << /Names [] >> >>"), domain.UnsupportedContent},
		{"media", "application/pdf", pdfDocument(1, "/RichMediaContent << >>"), domain.UnsupportedContent},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			id := uuid.NewString()
			requireStatus(t, f.upload(f.ta, id, f.accountID, c.media, c.data), 200)
			if err := f.service.ProcessNext(testContext); err != nil {
				t.Fatal(err)
			}
			a := f.metadata(f.ta, id)
			if a.State != domain.Rejected || a.Reason != c.reason {
				t.Fatalf("state=%s reason=%s expected=%s", a.State, a.Reason, c.reason)
			}
			requireStatus(t, f.request(f.tb, "GET", "/attachments/"+id+"/content", nil, ""), 409)
			if _, _, err := f.service.ReadForAnalysis(testContext, f.a, id); err == nil {
				t.Fatal("AI read rejected document")
			}
		})
	}
	requireStatus(t, f.upload(f.ta, uuid.NewString(), f.accountID, "image/png", make([]byte, domain.MaxBytes+1)), 400)
	requireStatus(t, f.upload(f.ta, uuid.NewString(), f.foreignAccount, "image/png", imageDocument(t, "image/png")), 404)
	requireStatus(t, f.upload(f.ta, "../other", f.accountID, "image/png", imageDocument(t, "image/png")), 400)
}
func TestSessionAndCSRFAtAttachmentBoundary(t *testing.T) {
	f := newFixture(t)
	id := uuid.NewString()
	data := imageDocument(t, "image/png")
	requireStatus(t, f.upload(f.ta, id, f.accountID, "image/png", data), 200)
	request := func(origin, csrf string) *http.Request {
		r, _ := http.NewRequest("POST", "http://localhost/api/v1/attachments", bytes.NewReader(nil))
		r.AddCookie(&http.Cookie{Name: "want_keep_session", Value: string(f.ta)})
		r.Header.Set("Origin", origin)
		r.Header.Set("X-CSRF-Token", csrf)
		return r
	}
	for _, r := range []*http.Request{request("http://evil.test", f.ta.CSRF()), request("http://localhost", f.tb.CSRF())} {
		w := httptest.NewRecorder()
		f.handler.ServeHTTP(w, r)
		if w.Code == 200 {
			t.Fatal("CSRF bypass")
		}
	}
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.identity_sessions SET created_at=clock_timestamp()-INTERVAL '32 minutes',last_activity_at=clock_timestamp()-INTERVAL '31 minutes' WHERE token_hash=$1`, f.ta.Hash()); err != nil {
		t.Fatal(err)
	}
	requireStatus(t, f.request(f.ta, "GET", "/attachments/"+id, nil, ""), 401)
	requireStatus(t, f.request(f.tb, "GET", "/attachments/"+id, nil, ""), 200)
}

func TestDimensionsExactUploadLimitAndEncryptedPDF(t *testing.T) {
	f := newFixture(t)
	pngData := imageDocument(t, "image/png")
	oversized := append([]byte{}, pngData...)
	binary.BigEndian.PutUint32(oversized[16:20], 6001)
	binary.BigEndian.PutUint32(oversized[20:24], 6667)
	binary.BigEndian.PutUint32(oversized[29:33], crc32.ChecksumIEEE(oversized[12:29]))
	id := uuid.NewString()
	requireStatus(t, f.upload(f.ta, id, f.accountID, "image/png", oversized), 200)
	if err := f.service.ProcessNext(testContext); err != nil {
		t.Fatal(err)
	}
	if f.metadata(f.ta, id).Reason != domain.LimitExceeded {
		t.Fatal("oversized dimensions not rejected")
	}
	dir := t.TempDir()
	input := filepath.Join(dir, "plain.pdf")
	output := filepath.Join(dir, "encrypted.pdf")
	if err := os.WriteFile(input, pdfDocument(1, ""), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("/usr/local/bin/qpdf", "--encrypt", "synthetic-user", "synthetic-owner", "256", "--", input, output)
	if err := cmd.Run(); err != nil {
		t.Fatal("encrypted fixture", err)
	}
	encrypted, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	id = uuid.NewString()
	requireStatus(t, f.upload(f.ta, id, f.accountID, "application/pdf", encrypted), 200)
	if err = f.service.ProcessNext(testContext); err != nil {
		t.Fatal(err)
	}
	if f.metadata(f.ta, id).State != domain.Rejected {
		t.Fatal("encrypted PDF accepted")
	}
	// Exact byte limit is admitted to validation; malformed content is still rejected there.
	id = uuid.NewString()
	requireStatus(t, f.upload(f.ta, id, f.accountID, "image/png", make([]byte, domain.MaxBytes)), 200)
	if err = f.service.ProcessNext(testContext); err != nil {
		t.Fatal(err)
	}
	if f.metadata(f.ta, id).State != domain.Rejected {
		t.Fatal("size boundary bypassed content validation")
	}
}
