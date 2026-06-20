package report

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var reportExt = map[string]bool{
	".html": true, ".htm": true, ".svg": true, ".png": true, ".jpg": true, ".jpeg": true,
}

func isReport(name string) bool { return reportExt[strings.ToLower(filepath.Ext(name))] }

func isImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".svg", ".png", ".jpg", ".jpeg":
		return true
	}
	return false
}

// ServeDir lists the reports (.html and images) in dir as a picker gallery,
// serves it on a loopback port, and opens the browser. Each entry opens in a new
// tab; images preview inline as thumbnails.
func ServeDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var reports []string
	for _, e := range entries {
		if !e.IsDir() && isReport(e.Name()) {
			reports = append(reports, e.Name())
		}
	}
	sort.Strings(reports)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	addr := "http://" + ln.Addr().String() + "/"
	page := []byte(gallery(dir, reports))
	mux := http.NewServeMux()
	mux.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(dir))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(page)
	})
	go openBrowser(addr)
	fmt.Printf("Plume is viewing %d report(s) at %s  (press Ctrl+C to stop)\n", len(reports), addr)
	return http.Serve(ln, mux)
}

func gallery(dir string, reports []string) string {
	var cards strings.Builder
	for _, name := range reports {
		href := "/files/" + url.PathEscape(name)
		if isImage(name) {
			fmt.Fprintf(&cards, `<a class="card" href="%s" target="_blank"><div class="thumb"><img src="%s" loading="lazy" alt=""></div><div class="name">%s</div></a>`, href, href, esc(name))
		} else {
			fmt.Fprintf(&cards, `<a class="card" href="%s" target="_blank"><div class="thumb html"><span>HTML</span></div><div class="name">%s</div></a>`, href, esc(name))
		}
	}
	body := cards.String()
	if body == "" {
		body = `<div class="empty">No reports in this folder. Generate one with <code>plume --out report.html .</code></div>`
	}
	return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Plume: reports</title><style>
:root{--bg:#0b0f17;--panel:#111826;--line:#1f2a3a;--ink:#e5edf7;--muted:#8aa0b8;--accent:#34d399}
*{box-sizing:border-box}html,body{margin:0;background:var(--bg);color:var(--ink);font:14px/1.4 ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,sans-serif}
header{padding:16px 20px;border-bottom:1px solid var(--line);display:flex;align-items:baseline;gap:12px;flex-wrap:wrap}
header .brand{font-weight:700;letter-spacing:.3px}header .brand span{color:var(--accent)}
header .sub{color:var(--muted);font-size:12px;word-break:break-all}
.grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(220px,1fr));gap:16px;padding:20px}
.card{display:block;background:var(--panel);border:1px solid var(--line);border-radius:12px;overflow:hidden;text-decoration:none;color:var(--ink);transition:border-color .15s,transform .15s}
.card:hover{border-color:#34507a;transform:translateY(-2px)}
.thumb{height:140px;display:flex;align-items:center;justify-content:center;background:#0c1320;overflow:hidden}
.thumb img{max-width:100%;max-height:100%;object-fit:contain}
.thumb.html span{font-weight:700;letter-spacing:1px;color:var(--muted);border:1px dashed var(--line);padding:10px 16px;border-radius:8px}
.name{padding:10px 12px;font-size:12.5px;border-top:1px solid var(--line);word-break:break-all}
.empty{padding:60px 20px;text-align:center;color:var(--muted)}
.empty code{color:var(--ink);background:#0c1320;padding:2px 6px;border-radius:6px}
</style></head><body><header><div class="brand">Plume<span>.</span></div><div class="sub">` + esc(dir) + `</div></header><div class="grid">` + body + `</div></body></html>`
}
