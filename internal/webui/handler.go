// Package webui serves a Fossil web interface (timeline, directory browsing,
// file viewing) alongside the xfer sync endpoint.
package webui

import (
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/danmestas/libfossil"
	"github.com/danmestas/libfossil/internal/blob"
	"github.com/danmestas/libfossil/internal/branch"
	"github.com/danmestas/libfossil/internal/content"
	"github.com/danmestas/libfossil/internal/fsltype"
	"github.com/danmestas/libfossil/internal/sync"
)

// Handler returns an http.Handler that serves the Fossil web UI on UI routes
// and delegates xfer sync requests to the xfer handler.
func Handler(r *libfossil.Repo) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/timeline", timelineHandler(r))
	mux.HandleFunc("/dir", dirHandler(r))
	mux.HandleFunc("/file", fileHandler(r))
	mux.HandleFunc("/info", infoHandler(r))
	mux.HandleFunc("/branches", branchesHandler(r))
	mux.HandleFunc("/leaves", leavesHandler(r))
	mux.HandleFunc("/raw", rawHandler(r))
	mux.HandleFunc("/blame", blameHandler(r))
	mux.HandleFunc("/annotate", blameHandler(r))  // alias
	mux.HandleFunc("/artifact", artifactHandler(r))
	mux.HandleFunc("/tree", treeHandler(r))
	mux.HandleFunc("/search", searchHandler(r))
	mux.HandleFunc("/fdiff", fdiffHandler(r))
	mux.HandleFunc("/vdiff", vdiffHandler(r))
	mux.HandleFunc("/users", usersHandler(r))
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodPost {
			inner := r.Inner()
			sync.XferHandler(inner, sync.HandleSync)(w, req)
			return
		}
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		homeHandler(r)(w, req)
	})
	return mux
}

// ServeBlocks starts the web UI server and blocks until ctx is cancelled.
func ServeBlocks(ctx interface {
	Done() <-chan struct{}
	Deadline() (time.Time, bool)
	Value(any) any
	Err() error
}, addr string, r *libfossil.Repo) error {
	if addr == "" {
		panic("webui.ServeBlocks: addr must not be empty")
	}
	if r == nil {
		panic("webui.ServeBlocks: r must not be nil")
	}
	srv := &http.Server{Handler: Handler(r)}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("webui: listen: %w", err)
	}
	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	err = srv.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// ---------- helpers ----------

// resolveCI resolves a ci parameter (RID number, UUID, or named version) to an RID.
func resolveCI(r *libfossil.Repo, ci string) (int64, error) {
	if rid, err := strconv.ParseInt(ci, 10, 64); err == nil && rid > 0 {
		if _, err := r.UUIDFromRID(rid); err == nil {
			return rid, nil
		}
	}
	return r.ResolveVersion(ci)
}

// versionUUID returns the full UUID for a ci/RID, or "tip" if unavailable.
func versionUUID(r *libfossil.Repo, rid int64, ci string) string {
	if uuid, err := r.UUIDFromRID(rid); err == nil {
		return uuid
	}
	return ci
}

// ---------- timeline ----------

func timelineHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		limit := 50
		if l := req.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		tip, err := r.ResolveVersion("")
		if err != nil {
			render(w, timelinePage, timelineData{Entries: nil})
			return
		}
		entries, err := r.Timeline(libfossil.LogOpts{Start: tip, Limit: limit})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		render(w, timelinePage, timelineData{Entries: entries})
	}
}

type timelineData struct {
	Entries []libfossil.LogEntry
}

// ---------- directory listing ----------

func dirHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ci := req.URL.Query().Get("ci")
		if ci == "" {
			ci = "tip"
		}
		rid, err := resolveCI(r, ci)
		if err != nil {
			http.Error(w, "version not found: "+err.Error(), http.StatusNotFound)
			return
		}
		uuid := versionUUID(r, rid, ci)
		files, err := r.ListFiles(rid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		render(w, dirPage, dirData{UUID: uuid, Files: files})
	}
}

type dirData struct {
	UUID  string
	Files []libfossil.FileEntry
}

// ---------- file content ----------

func fileHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Query().Get("name")
		if path == "" {
			http.Error(w, "missing name parameter", http.StatusBadRequest)
			return
		}
		ci := req.URL.Query().Get("ci")
		if ci == "" {
			ci = "tip"
		}
		rid, err := resolveCI(r, ci)
		if err != nil {
			http.Error(w, "version not found: "+err.Error(), http.StatusNotFound)
			return
		}
		data, err := r.ReadFile(rid, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		ver := versionUUID(r, rid, ci)
		render(w, filePage, fileData{Path: path, Version: ver, Body: string(data)})
	}
}

type fileData struct {
	Path    string
	Version string
	Body    string
}

// ---------- info ----------

func infoHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ci := req.URL.Query().Get("ci")
		if ci == "" {
			ci = "tip"
		}
		rid, err := resolveCI(r, ci)
		if err != nil {
			http.Error(w, "version not found: "+err.Error(), http.StatusNotFound)
			return
		}
		entries, err := r.Timeline(libfossil.LogOpts{Start: rid, Limit: 1})
		if err != nil || len(entries) == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		files, err := r.ListFiles(rid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		render(w, infoPage, infoData{Entry: entries[0], Files: files})
	}
}

type infoData struct {
	Entry libfossil.LogEntry
	Files []libfossil.FileEntry
}

// ---------- branches ----------

func branchesHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		branches, err := branch.List(r.Inner())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		render(w, branchesPage, branchesData{Branches: branches})
	}
}

type branchesData struct {
	Branches []branch.Branch
}

// ---------- leaves ----------

func leavesHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		type leafEntry struct {
			UUID    string
			Comment string
			Time    time.Time
		}
		rows, err := r.DB().Query(`
			SELECT b.uuid, COALESCE(e.comment,''), e.mtime
			FROM leaf l
			JOIN blob b ON l.rid=b.rid
			JOIN event e ON e.objid=l.rid
			ORDER BY e.mtime DESC
		`)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var leaves []leafEntry
		for rows.Next() {
			var le leafEntry
			if err := rows.Scan(&le.UUID, &le.Comment, &le.Time); err != nil {
				break
			}
			leaves = append(leaves, le)
		}
		render(w, leavesPage, leavesData{Leaves: leaves})
	}
}

type leavesData struct {
	Leaves interface{}
}

// ---------- raw file ----------

func rawHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Query().Get("name")
		if path == "" {
			http.Error(w, "missing name parameter", http.StatusBadRequest)
			return
		}
		ci := req.URL.Query().Get("ci")
		if ci == "" {
			ci = "tip"
		}
		rid, err := resolveCI(r, ci)
		if err != nil {
			http.Error(w, "version not found", http.StatusNotFound)
			return
		}
		data, err := r.ReadFile(rid, path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(data)
	}
}

// ---------- blame / annotate ----------

func blameHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Query().Get("name")
		if path == "" {
			http.Error(w, "missing name parameter", http.StatusBadRequest)
			return
		}
		ci := req.URL.Query().Get("ci")
		if ci == "" {
			ci = "tip"
		}
		rid, err := resolveCI(r, ci)
		if err != nil {
			http.Error(w, "version not found", http.StatusNotFound)
			return
		}
		lines, err := r.Annotate(libfossil.AnnotateOpts{FilePath: path, StartRID: rid})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ver := versionUUID(r, rid, ci)
		render(w, blamePage, blameData{Path: path, Version: ver, Lines: lines})
	}
}

type blameData struct {
	Path    string
	Version string
	Lines   []libfossil.AnnotatedLine
}

// ---------- artifact ----------

func artifactHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		id := req.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "missing id parameter", http.StatusBadRequest)
			return
		}
		// Resolve id: try as UUID first, then as RID.
		blobFslID, ok := blob.Exists(r.DB(), id)
		if !ok {
			rid, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				http.Error(w, "artifact not found", http.StatusNotFound)
				return
			}
			blobFslID = fsltype.FslID(rid)
		}
		uuid, err := r.UUIDFromRID(int64(blobFslID))
		if err != nil {
			uuid = id
		}
		data, err := content.Expand(r.DB(), blobFslID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Artifact-UUID", uuid)
		w.Write(data)
	}
}

// ---------- home ----------

func homeHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		db := r.DB()
		var commitCount, branchCount, userCount int
		db.QueryRow("SELECT COUNT(*) FROM event WHERE type='ci'").Scan(&commitCount)
		db.QueryRow("SELECT COUNT(*) FROM tagxref WHERE tagid IN (SELECT tagid FROM tag WHERE tagname='branch') AND tagtype>0").Scan(&branchCount)
		db.QueryRow("SELECT COUNT(*) FROM user").Scan(&userCount)

		type recentEntry struct {
			UUID    string
			Comment string
			User    string
			Time    time.Time
		}
		rows, err := db.Query(`
			SELECT b.uuid, e.comment, e.user, e.mtime
			FROM event e JOIN blob b ON b.rid=e.objid
			WHERE e.type='ci'
			ORDER BY e.mtime DESC LIMIT 5
		`)
		var recent []recentEntry
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r recentEntry
				if err := rows.Scan(&r.UUID, &r.Comment, &r.User, &r.Time); err != nil {
					break
				}
				recent = append(recent, r)
			}
		}
		render(w, homePage, homeData{
			CommitCount: commitCount,
			BranchCount: branchCount,
			UserCount:   userCount,
			Recent:      recent,
		})
	}
}

type homeData struct {
	CommitCount int
	BranchCount int
	UserCount   int
	Recent      interface{}
}

// ---------- tree (alias for dir) ----------

func treeHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ci := req.URL.Query().Get("ci")
		if ci == "" {
			ci = "tip"
		}
		http.Redirect(w, req, "/dir?ci="+ci, http.StatusFound)
	}
}

// ---------- search ----------

func searchHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		q := req.URL.Query().Get("q")
		if q == "" {
			render(w, searchPage, searchData{Query: "", Results: nil})
			return
		}
		type sr struct {
			UUID    string
			Comment string
			User    string
			Time    time.Time
		}
		rows, err := r.DB().Query(`
			SELECT b.uuid, e.comment, e.user, e.mtime
			FROM event e
			JOIN blob b ON b.rid=e.objid
			WHERE e.type='ci' AND e.comment LIKE '%' || ? || '%'
			ORDER BY e.mtime DESC
			LIMIT 100
		`, q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var results []sr
		for rows.Next() {
			var r sr
			if err := rows.Scan(&r.UUID, &r.Comment, &r.User, &r.Time); err != nil {
				break
			}
			results = append(results, r)
		}
		render(w, searchPage, searchData{Query: q, Results: results})
	}
}

type searchData struct {
	Query   string
	Results interface{}
}

// ---------- file diff ----------

func fdiffHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ci := req.URL.Query().Get("ci")
		if ci == "" {
			ci = "tip"
		}
		name := req.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "missing name parameter", http.StatusBadRequest)
			return
		}
		rid, err := resolveCI(r, ci)
		if err != nil {
			http.Error(w, "version not found", http.StatusNotFound)
			return
		}
		ver := versionUUID(r, rid, ci)
		entries, err := r.Timeline(libfossil.LogOpts{Start: rid, Limit: 1})
		if err != nil || len(entries) == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		// Resolve first parent RID.
		parentRID := int64(0)
		if len(entries[0].Parents) > 0 {
			parentRID, _ = r.ResolveVersion(entries[0].Parents[0])
		}
		if parentRID == 0 {
			// No parent — show the file as-is.
			data, err := r.ReadFile(rid, name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			render(w, diffPage, diffData{Name: name, Version: ver, Unified: string(data)})
			return
		}
		diffs, err := r.Diff(parentRID, rid, name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		unified := ""
		if len(diffs) > 0 {
			unified = diffs[0].Unified
		}
		render(w, diffPage, diffData{Name: name, Version: ver, Unified: unified})
	}
}

// ---------- version diff ----------

func vdiffHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ci := req.URL.Query().Get("ci")
		if ci == "" {
			ci = "tip"
		}
		rid, err := resolveCI(r, ci)
		if err != nil {
			http.Error(w, "version not found", http.StatusNotFound)
			return
		}
		ver := versionUUID(r, rid, ci)
		entries, err := r.Timeline(libfossil.LogOpts{Start: rid, Limit: 1})
		if err != nil || len(entries) == 0 {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		parentRID := int64(0)
		if len(entries[0].Parents) > 0 {
			parentRID, _ = r.ResolveVersion(entries[0].Parents[0])
		}
		if parentRID == 0 {
			render(w, diffPage, diffData{Name: "(new commit)", Version: ver, Unified: "No parent to diff against."})
			return
		}
		diffs, err := r.Diff(parentRID, rid, "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		render(w, diffPage, diffData{Name: fmt.Sprintf("Changes (%d files)", len(diffs)), Version: ver, Unified: formatVDiff(diffs)})
	}
}

func formatVDiff(diffs []libfossil.DiffEntry) string {
	var out string
	for _, d := range diffs {
		out += "=== " + d.Name + " ===\n"
		if d.Unified != "" {
			out += d.Unified
		}
		out += "\n"
	}
	return out
}

// ---------- users ----------

func usersHandler(r *libfossil.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		users, err := r.ListUsers()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		render(w, usersPage, usersData{Users: users})
	}
}

type usersData struct {
	Users []libfossil.User
}

// ---------- rendering ----------

type diffData struct {
	Name     string
	Version  string
	Unified  string
}

var (
	homePage      = newTemplate("home", pageLayout+homeHTML)
	timelinePage  = newTemplate("timeline", pageLayout+timelineHTML)
	dirPage       = newTemplate("dir", pageLayout+dirHTML)
	filePage      = newTemplate("file", pageLayout+fileHTML)
	infoPage      = newTemplate("info", pageLayout+infoHTML)
	branchesPage  = newTemplate("branches", pageLayout+branchesHTML)
	leavesPage    = newTemplate("leaves", pageLayout+leavesHTML)
	blamePage     = newTemplate("blame", pageLayout+blameHTML)
	searchPage    = newTemplate("search", pageLayout+searchHTML)
	diffPage      = newTemplate("diff", pageLayout+diffHTML)
	usersPage     = newTemplate("users", pageLayout+usersHTML)
)

func newTemplate(name, text string) *template.Template {
	return template.Must(template.New(name).Funcs(template.FuncMap{
		"shortUUID": func(uuid string) string {
			if len(uuid) > 10 {
				return uuid[:10]
			}
			return uuid
		},
		"timeFmt": func(t time.Time) string {
			return t.Format("2006-01-02 15:04")
		},
		"add": func(a, b int) int {
			return a + b
		},
	}).Parse(text))
}

func render(w http.ResponseWriter, t *template.Template, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(w, data); err != nil {
		slog.Error("webui: render", "err", err)
	}
}

const pageLayout = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Fossil</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; color: #333; }
a { color: #2563eb; text-decoration: none; }
a:hover { text-decoration: underline; }
.container { max-width: 960px; margin: 0 auto; background: #fff; border-radius: 8px; padding: 24px; box-shadow: 0 1px 3px rgba(0,0,0,.1); }
h1 { margin-top: 0; border-bottom: 2px solid #e5e7eb; padding-bottom: 8px; font-size: 1.3em; }
.entry { padding: 10px 0; border-bottom: 1px solid #f0f0f0; }
.entry:last-child { border-bottom: none; }
.uuid { font-family: "SF Mono", Monaco, monospace; font-size: .85em; color: #666; }
.meta { font-size: .85em; color: #666; margin-top: 2px; }
pre { background: #f8f9fa; border-radius: 4px; padding: 12px; overflow-x: auto; font-size: .9em; }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 6px 8px; border-bottom: 1px solid #f0f0f0; }
th { font-weight: 600; color: #666; font-size: .85em; text-transform: uppercase; }
.nav { margin-bottom: 16px; font-size: .9em; }
.nav a { margin-right: 12px; }
.navbar { background: #2563eb; color: #fff; padding: 10px 20px; margin: -24px -24px 16px -24px; border-radius: 8px 8px 0 0; }
.navbar a { color: #fff; margin-right: 16px; font-size: .9em; }
.navbar a:hover { text-decoration: underline; }
.navbar .title { font-weight: 600; margin-right: 20px; }
.cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px,1fr)); gap: 12px; }
.card { background: #f8f9fa; border-radius: 6px; padding: 16px; border: 1px solid #e5e7eb; }
.card h3 { margin: 0 0 4px 0; font-size: 1em; }
.card h3 a { color: inherit; }
.card p { margin: 0; font-size: .85em; color: #666; }
</style>
</head>
<body>
<div class="container">
<div class="navbar"><span class="title"><a href="/">Fossil</a></span><a href="/timeline">Timeline</a><a href="/branches">Branches</a><a href="/leaves">Leaves</a><a href="/search">Search</a><a href="/users">Users</a><a href="/dir">Files</a></div>
`

// ---------- timeline HTML ----------

const timelineHTML = `
<h1>Timeline</h1>
<p style="margin-top:-8px;color:#666;font-size:.9em">Commit history — newest first</p>
{{range .Entries}}
<div class="entry">
  <strong><a href="/info?ci={{.UUID}}">{{.Comment}}</a></strong>
  <div class="meta">{{.User}} &middot; {{.Time.Format "2006-01-02 15:04"}} &middot; <span class="uuid">{{shortUUID .UUID}}</span></div>
</div>
{{else}}
<p>No commits yet.</p>
{{end}}
`

// ---------- dir HTML ----------

const dirHTML = `
<h1>Browse — <span class="uuid">{{.UUID}}</span></h1>
<p style="margin-top:-8px;color:#666;font-size:.9em">Files in this version — click a file to view</p>
<table>
<tr><th>Name</th><th></th></tr>
{{range .Files}}
<tr>
  <td><a href="/file?ci={{$.UUID}}&name={{.Name}}">{{.Name}}</a></td>
  <td class="uuid">{{shortUUID .UUID}}</td>
</tr>
{{else}}
<tr><td colspan="2">No files.</td></tr>
{{end}}
</table>
`

// ---------- file HTML ----------

const fileHTML = `
<h1>{{.Path}}</h1>
<p style="margin-top:-8px;color:#666;font-size:.9em">File content — <a href="/blame?ci={{.Version}}&name={{.Path}}">Blame</a> &middot; <a href="/fdiff?ci={{.Version}}&name={{.Path}}">Diff</a> &middot; <a href="/raw?ci={{.Version}}&name={{.Path}}">Raw</a></p>
<div class="nav"><a href="/dir?ci={{.Version}}">Back</a> &middot; <a href="/blame?ci={{.Version}}&name={{.Path}}">Blame</a> &middot; <a href="/raw?ci={{.Version}}&name={{.Path}}">Raw</a></div>
<pre>{{.Body}}</pre>
`

// ---------- info HTML ----------

const infoHTML = `
<h1>Commit — <span class="uuid">{{.Entry.UUID}}</span></h1>
<p style="margin-top:-8px;color:#666;font-size:.9em">Commit details — <a href="/vdiff?ci={{.Entry.UUID}}">View diff</a> &middot; <a href="/dir?ci={{.Entry.UUID}}">Browse files</a></p>
<div class="meta">
  <p><strong>User:</strong> {{.Entry.User}}<br>
  <strong>Date:</strong> {{.Entry.Time.Format "2006-01-02 15:04:05"}}<br>
  <strong>Parents:</strong> {{range .Entry.Parents}}<span class="uuid">{{shortUUID .}}</span> {{else}}none{{end}}</p>
</div>
<pre>{{.Entry.Comment}}</pre>
<table>
<tr><th>File</th><th></th></tr>
{{range .Files}}
<tr>
  <td><a href="/file?ci={{$.Entry.UUID}}&name={{.Name}}">{{.Name}}</a></td>
  <td class="uuid">{{shortUUID .UUID}}</td>
</tr>
{{else}}
<tr><td colspan="2">No files.</td></tr>
{{end}}
</table>
`

// ---------- branches HTML ----------

const branchesHTML = `
<h1>Branches</h1>
<p style="margin-top:-8px;color:#666;font-size:.9em">All branches — closed branches are struck through</p>
<table>
<tr><th>Name</th><th>Checkins</th><th>Latest</th></tr>
{{range .Branches}}
<tr>
  <td>{{if .IsClosed}}<s>{{.Name}}</s>{{else}}<a href="/timeline?branch={{.Name}}">{{.Name}}</a>{{end}}</td>
  <td>{{.CheckinCount}}</td>
  <td class="uuid"><a href="/info?ci={{.LatestUUID}}">{{shortUUID .LatestUUID}}</a></td>
</tr>
{{else}}
<tr><td colspan="3">No branches.</td></tr>
{{end}}
</table>
`

// ---------- leaves HTML ----------

const leavesHTML = `
<h1>Leaves</h1>
<p style="margin-top:-8px;color:#666;font-size:.9em">Leaf commits — checkins with no children</p>
{{range .Leaves}}
<div class="entry">
  <a href="/info?ci={{.UUID}}"><strong>{{.Comment}}</strong></a>
  <div class="meta">{{.Time.Format "2006-01-02 15:04"}} &middot; <span class="uuid">{{shortUUID .UUID}}</span></div>
</div>
{{else}}
<p>No leaves.</p>
{{end}}
`

// ---------- blame HTML ----------

const blameHTML = `
<h1>Blame — {{.Path}}</h1>
<p style="margin-top:-8px;color:#666;font-size:.9em">Each line annotated with its last-modified commit</p>
<div class="nav"><a href="/file?ci={{.Version}}&name={{.Path}}">Back to file</a></div>
<table>
<tr><th>Line</th><th>Source</th><th>Content</th></tr>
{{range $i, $l := .Lines}}
<tr>
  <td style="color:#999;text-align:right;padding-right:8px">{{add $i 1}}</td>
  <td class="uuid"><a href="/info?ci={{$l.UUID}}">{{shortUUID $l.UUID}}</a></td>
  <td><pre style="margin:0;padding:2px 4px;background:none">{{$l.Text}}</pre></td>
</tr>
{{else}}
<tr><td colspan="3">No content.</td></tr>
{{end}}
</table>
`

// ---------- home HTML ----------

const homeHTML = `
<h1>Repository Overview</h1>
<div class="cards">
  <div class="card"><h3><a href="/timeline">{{.CommitCount}}</a></h3><p>Commits</p></div>
  <div class="card"><h3><a href="/branches">{{.BranchCount}}</a></h3><p>Branches</p></div>
  <div class="card"><h3><a href="/users">{{.UserCount}}</a></h3><p>Users</p></div>
</div>
{{if .Recent}}
<h2 style="font-size:1em;margin-top:20px">Recent Activity</h2>
{{range .Recent}}
<div class="entry">
  <a href="/info?ci={{.UUID}}"><strong>{{.Comment}}</strong></a>
  <div class="meta">{{.User}} &middot; {{.Time.Format "2006-01-02 15:04"}} &middot; <span class="uuid">{{shortUUID .UUID}}</span></div>
</div>
{{end}}
{{end}}
`

// ---------- search HTML ----------

const searchHTML = `
<h1>Search</h1>
<p style="margin-top:-8px;color:#666;font-size:.9em">Search commit comments by keyword</p>
<form action="/search" method="GET" style="margin-bottom:16px">
<input type="text" name="q" value="{{.Query}}" placeholder="Search comments..." style="padding:6px 10px;width:300px;border:1px solid #ccc;border-radius:4px">
<button type="submit" style="padding:6px 12px;background:#2563eb;color:#fff;border:none;border-radius:4px">Search</button>
</form>
{{if .Query}}
  {{range .Results}}
  <div class="entry">
    <a href="/info?ci={{.UUID}}"><strong>{{.Comment}}</strong></a>
    <div class="meta">{{.User}} &middot; {{.Time.Format "2006-01-02 15:04"}} &middot; <span class="uuid">{{shortUUID .UUID}}</span></div>
  </div>
  {{else}}
  <p>No results for "{{.Query}}".</p>
  {{end}}
{{end}}
`

// ---------- diff HTML ----------

const diffHTML = `
<h1>{{.Name}}</h1>
<p style="margin-top:-8px;color:#666;font-size:.9em">Unified diff — green lines are additions, red are removals</p>
<div class="nav"><a href="/info?ci={{.Version}}">Back to commit</a></div>
<pre>{{.Unified}}</pre>
`

// ---------- users HTML ----------

const usersHTML = `
<h1>Users</h1>
<p style="margin-top:-8px;color:#666;font-size:.9em">Repository users and their capability flags</p>
<table>
<tr><th>Login</th><th>Capabilities</th></tr>
{{range .Users}}
<tr>
  <td>{{.Login}}</td>
  <td class="uuid">{{.Caps}}</td>
</tr>
{{else}}
<tr><td colspan="2">No users.</td></tr>
{{end}}
</table>
`
