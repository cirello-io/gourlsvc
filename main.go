// Command gourlsvc starts a very simple go-url service. See more at
package main // import "cirello.io/gourlsvc"

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"

	_ "github.com/mattn/go-sqlite3" // SQLite3 driver
)

func main() {
	log.SetPrefix("gourlsvc: ")
	log.SetFlags(0)
	db, err := sql.Open("sqlite3", "links.db")
	if err != nil {
		log.Fatal(err)
	}

	if err := migrate(db); err != nil {
		log.Fatal(err)
	}

	s := &server{db: db}
	s.registerRoutes()

	log.Println("starting server")
	log.Fatal(http.ListenAndServe(":8090", s.router))
}

func migrate(db *sql.DB) error {
	cmds := []string{
		`CREATE TABLE IF NOT EXISTS links ( name varchar(255), url varchar(255) );`,
		`CREATE UNIQUE INDEX IF NOT EXISTS links_name ON links (name);`,
	}
	for i, cmd := range cmds {
		_, err := db.Exec(cmd)
		if err != nil {
			return fmt.Errorf("cannot execute query %v: %w", i, err)
		}
	}
	return nil
}

type server struct {
	db     *sql.DB
	router *http.ServeMux
}

func (s *server) registerRoutes() {
	s.router = http.NewServeMux()
	s.router.HandleFunc("/edit/", s.edit)
	s.router.HandleFunc("/", s.root)
}

type Link struct {
	Name string
	URL  string
}

func (s *server) edit(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.Split(strings.TrimPrefix(r.URL.EscapedPath(), "/edit/"), "/")
	if len(urlParts) == 0 {
		http.NotFound(w, r)
		return
	}
	var (
		requestName = urlParts[0]
		link        Link
	)
	row := s.db.QueryRowContext(r.Context(), "SELECT name, url FROM links WHERE name = $1", requestName)
	if err := row.Scan(&link.Name, &link.URL); err == sql.ErrNoRows {
		link.Name = requestName
	} else if err != nil && err != sql.ErrNoRows {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		name := r.FormValue("name")
		url := r.FormValue("url")

		_, err := s.db.Exec("INSERT OR REPLACE INTO links values ($1, $2)", name, url)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	var readOnly string
	if link.Name != "" {
		readOnly = "readonly"
	}
	editForm.Execute(w, struct {
		ReadOnly string
		Link     Link
	}{readOnly, link})
}

func (s *server) root(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.EscapedPath(), "/")
	if name == "" {
		s.list(w, r)
		return
	}
	s.redirect(w, r)
}

func (s *server) list(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.QueryContext(r.Context(), "SELECT name, url FROM links ORDER BY name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var links []Link
	for rows.Next() {
		var link Link
		if err := rows.Scan(&link.Name, &link.URL); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		links = append(links, link)
	}

	listLinks.Execute(w, links)
}

func (s *server) redirect(w http.ResponseWriter, r *http.Request) {
	{
		name := strings.TrimPrefix(r.URL.EscapedPath(), "/")
		var url string
		row := s.db.QueryRowContext(r.Context(), "SELECT url FROM links WHERE name = $1", name)
		if err := row.Scan(&url); err == nil && url != "" {
			http.Redirect(w, r, url, http.StatusSeeOther)
			return
		}
	}
	{
		urlParts := strings.Split(r.URL.EscapedPath(), "/")
		if len(urlParts) == 1 {
			http.NotFound(w, r)
			return
		}
		name := urlParts[1]
		rest := path.Clean(strings.TrimPrefix(r.URL.EscapedPath(), "/"+name))
		var url string
		row := s.db.QueryRowContext(r.Context(), "SELECT url FROM links WHERE name = $1", name)
		if err := row.Scan(&url); err == sql.ErrNoRows {
			http.NotFound(w, r)
			return
		} else if err != nil && err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if rest != "." {
			url += rest
		}
		http.Redirect(w, r, url, http.StatusSeeOther)
	}
}

var listLinks = template.Must(template.New("listLinks").Parse(`<!doctype html>
<html lang="en">
<head>
<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
</head>
<body>
<table>
	<thead>
		<tr>
			<th>Name</th>
			<th>URL</th>
		</tr>
	</thead>
	<tbody>
{{ range . }}
	<tr>
		<td>{{ .Name }}</th>
		<td><a href="{{ .URL }}">{{ .URL }}</a></th>
	</tr>
{{ end }}
	</tbody>
</table>
</body>
</html>`))

var editForm = template.Must(template.New("editForm").Parse(`<!doctype html>
<html lang="en">
<head>
<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
</head>
<body>
<form method="POST">
name:<input type="text" size="50" name="name" {{ .ReadOnly }} value="{{- .Link.Name -}}"/><br/>
url:<input type="text" size="120" name="url" value="{{- .Link.URL -}}"/><br/>
<input type="submit"/>
</form>
</body>
</html>`))
