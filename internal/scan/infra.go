package scan

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/judahpaul16/plume/internal/catalog"
	"github.com/judahpaul16/plume/internal/graph"
	"github.com/odvcencio/gotreesitter/grammars"
)

// isInfra reports whether a file is infrastructure-as-code rather than app code.
func isInfra(entry *grammars.LangEntry, path string) bool {
	base := strings.ToLower(filepath.Base(path))
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case ext == ".tf" || ext == ".tfvars":
		return true
	case strings.HasPrefix(base, "docker-compose") || strings.HasPrefix(base, "compose."):
		return true
	case strings.HasPrefix(base, "serverless."):
		return true
	case base == "chart.yaml" || base == "values.yaml":
		return true
	}
	if entry != nil && (entry.Name == "hcl" || entry.Name == "terraform") {
		return true
	}
	if ext == ".yaml" || ext == ".yml" {
		p := strings.ToLower(path)
		for _, m := range []string{"k8s", "kubernetes", "manifest", "/deploy", "chart", "helm"} {
			if strings.Contains(p, m) {
				return true
			}
		}
	}
	return false
}

var wordRe = regexp.MustCompile(`[a-zA-Z0-9_]+`)

// enrichFromInfra reads infra-as-code and makes generic code-detected stores
// concrete when the IaC declares the backing resource.
func enrichFromInfra(g *graph.Graph, src string) {
	for _, tok := range wordRe.FindAllString(strings.ToLower(src), -1) {
		if r, ok := catalog.ByToken[tok]; ok {
			g.Relabel(r.EnrichID, r.Label)
		}
	}
}
