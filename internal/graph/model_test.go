package graph

import "testing"

func TestGraphDedupAndPrune(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "a", Kind: Source})
	g.AddNode(Node{ID: "b", Kind: Service})
	g.AddNode(Node{ID: "orphan", Kind: Store})
	g.AddFlow(Flow{From: "a", To: "b", Categories: []string{"email"}})
	g.AddFlow(Flow{From: "a", To: "b", Categories: []string{"name"}})
	g.Finalize()

	if len(g.Flows) != 1 {
		t.Fatalf("flows = %d, want 1 merged", len(g.Flows))
	}
	if got := g.Flows[0].Categories; len(got) != 2 {
		t.Errorf("merged categories = %v, want two", got)
	}
	for _, n := range g.Nodes {
		if n.ID == "orphan" {
			t.Error("orphan node with no flows was not pruned")
		}
	}
}

func TestRelabel(t *testing.T) {
	g := New()
	g.AddNode(Node{ID: "store:db", Kind: Store, Label: "Database"})
	g.Relabel("store:db", "PostgreSQL")
	g.Relabel("store:missing", "ignored")
	if g.Nodes[0].Label != "PostgreSQL" {
		t.Errorf("label = %q, want PostgreSQL", g.Nodes[0].Label)
	}
}
