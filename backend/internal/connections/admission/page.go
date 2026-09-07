package admission

import "sort"

// WithOmissions preserves gaps discovered by earlier pages or normalization inside the current transaction.
func (p Page) WithOmissions(omissions []string) Page {
	seen := map[string]bool{}
	for _, gap := range p.Gaps {
		seen[gap] = true
	}
	for _, gap := range omissions {
		seen[gap] = true
	}
	p.Gaps = make([]string, 0, len(seen))
	for gap := range seen {
		p.Gaps = append(p.Gaps, gap)
	}
	sort.Strings(p.Gaps)
	if len(p.Gaps) > 0 && p.Coverage == "complete" {
		p.Coverage = "partial"
	}
	return p
}
