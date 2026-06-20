// Package report renders a flow graph into a single self-contained interactive
// HTML page and serves it for viewing in the browser.
package report

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/judahpaul16/plume/internal/graph"
)

// Render inlines the viewer, the graph data, and the Go-computed node positions
// into one self-contained HTML document, so the page and the CLI image share one
// layout.
func Render(webFS fs.FS, g *graph.Graph) ([]byte, error) {
	idx, err := fs.ReadFile(webFS, "web/index.html")
	if err != nil {
		return nil, err
	}
	viewer, err := fs.ReadFile(webFS, "web/viewer.js")
	if err != nil {
		return nil, err
	}
	dataJSON, err := json.Marshal(g)
	if err != nil {
		return nil, err
	}
	type lnode struct {
		X   float64 `json:"x"`
		Y   float64 `json:"y"`
		W   float64 `json:"w"`
		H   float64 `json:"h"`
		Sub string  `json:"sub"`
	}
	d := layout(g)
	pos := make(map[string]lnode, len(d.Nodes))
	for _, n := range d.Nodes {
		pos[n.ID] = lnode{X: n.X, Y: n.Y, W: nodeW, H: nodeH, Sub: n.Kind}
	}
	layoutJSON, err := json.Marshal(pos)
	if err != nil {
		return nil, err
	}
	out := string(idx)
	out = strings.Replace(out, "/*__PLUME_DATA__*/", string(dataJSON), 1)
	out = strings.Replace(out, "/*__PLUME_LAYOUT__*/", string(layoutJSON), 1)
	out = strings.Replace(out, "/*__PLUME_VIEWER__*/", string(viewer), 1)
	return []byte(out), nil
}

// Serve hosts the HTML on a loopback port, opens the browser, and blocks. It
// serves over http (not file://) so the viewer's layout worker runs everywhere.
func Serve(html []byte) error { return ServeContent(html, "text/html; charset=utf-8") }

// ServeContent hosts arbitrary content on a loopback port, opens the browser,
// and blocks. SVG and PNG reports use it to display directly in the browser.
func ServeContent(data []byte, contentType string) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	url := "http://" + ln.Addr().String() + "/"
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(data)
	})
	go openBrowser(url)
	fmt.Printf("Plume is viewing at %s  (press Ctrl+C to stop)\n", url)
	return http.Serve(ln, mux)
}

// ContentType maps a report file extension to its MIME type for serving.
func ContentType(path string) string {
	p := strings.ToLower(path)
	switch {
	case strings.HasSuffix(p, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(p, ".png"):
		return "image/png"
	case strings.HasSuffix(p, ".jpg"), strings.HasSuffix(p, ".jpeg"):
		return "image/jpeg"
	default:
		return "text/html; charset=utf-8"
	}
}

func openBrowser(url string) {
	var c *exec.Cmd
	switch {
	case runtime.GOOS == "darwin":
		c = exec.Command("open", url)
	case runtime.GOOS == "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case isWSL():
		c = exec.Command("explorer.exe", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	_ = c.Start()
}

func isWSL() bool {
	b, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	v := strings.ToLower(string(b))
	return strings.Contains(v, "microsoft") || strings.Contains(v, "wsl")
}
