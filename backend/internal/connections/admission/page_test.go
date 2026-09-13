package admission

import (
	"fmt"
	"testing"
)

func TestPageOmissionsPreserveInputAndCoverage(t *testing.T) {
	p := Page{Coverage: "complete"}
	partial := p.WithOmissions([]string{"source_ambiguous", "unsupported_asset", "source_ambiguous"})
	if p.Coverage != "complete" || len(p.Gaps) != 0 || partial.Coverage != "partial" || len(partial.Gaps) != 2 {
		t.Fatal("coverage or immutability failed")
	}
	next := Page{Coverage: "complete"}.WithOmissions(partial.Gaps)
	if next.Coverage != "partial" || len(next.Gaps) != 2 {
		t.Fatal("prior page gaps lost")
	}
}

func TestPageRejectsCumulativeGapOverflow(t *testing.T) {
	gaps := make([]string, 100)
	for index := range gaps {
		gaps[index] = fmt.Sprintf("gap-%d", index)
	}
	page := Page{EvidenceRef: "synthetic", Coverage: "partial", Gaps: gaps}
	if page.Validate() != nil {
		t.Fatal("valid gap boundary was rejected")
	}
	page = page.WithOmissions([]string{"one-more-gap"})
	if page.Validate() == nil {
		t.Fatal("cumulative gap overflow was accepted")
	}
}
