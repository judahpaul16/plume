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

// Render inlines the embedded viewer and the graph data into one HTML document.
func Render(webFS fs.FS, g *graph.Graph) ([]byte, error) {
	idx, err := fs.ReadFile(webFS, "web/index.html")
	if err != nil {
		return nil, err
	}
	viewer, err := fs.ReadFile(webFS, "web/viewer.js")
	if err != nil {
		return nil, err
	}
	elkjs, err := fs.ReadFile(webFS, "web/elk.bundled.js")
	if err != nil {
		return nil, err
	}
	dataJSON, err := json.Marshal(g)
	if err != nil {
		return nil, err
	}
	out := string(idx)
	out = strings.Replace(out, "/*__PLUME_ELK__*/", string(elkjs), 1)
	out = strings.Replace(out, "/*__PLUME_DATA__*/", string(dataJSON), 1)
	out = strings.Replace(out, "/*__PLUME_VIEWER__*/", string(viewer), 1)
	return []byte(out), nil
}

// Serve hosts the HTML on a loopback port, opens the browser, and blocks. It
// serves over http (not file://) so the viewer's layout worker runs everywhere.
func Serve(html []byte) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	url := "http://" + ln.Addr().String() + "/"
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(html)
	})
	go openBrowser(url)
	fmt.Printf("Plume is viewing at %s  (press Ctrl+C to stop)\n", url)
	return http.Serve(ln, mux)
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
