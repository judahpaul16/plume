package scan

import "testing"

func TestScanSample(t *testing.T) {
	g, st := Scan([]string{"../../testdata/sample"})
	if st.Parsed < 3 {
		t.Fatalf("parsed %d files, want >= 3", st.Parsed)
	}
	has := map[string]bool{}
	for _, n := range g.Nodes {
		has[n.ID] = true
	}
	for _, id := range []string{"source:user", "ext:api.stripe.com", "ext:api.segment.io", "sink:log", "store:db", "store:cache"} {
		if !has[id] {
			t.Errorf("missing expected node %q", id)
		}
	}
	cats := map[string]bool{}
	for _, c := range g.Categories {
		cats[c.ID] = true
	}
	for _, c := range []string{"email", "card", "ssn", "name"} {
		if !cats[c] {
			t.Errorf("missing expected category %q", c)
		}
	}
	for _, n := range g.Nodes {
		if n.ID == "store:db" && n.Label != "PostgreSQL" {
			t.Errorf("store:db label = %q, want PostgreSQL from infra enrichment", n.Label)
		}
	}
}
