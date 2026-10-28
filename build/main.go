package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
)

var buildTime = "unknown"

// ── FORKING THIS TEMPLATE: change these two. ──

// defaultAppName is the fallback when APP is unset (local dev without .env), so
// the app still has a name and a database. In prod deploy.sh always writes APP.
const defaultAppName = "auth-client"

// readmeURL is the profile menu's "readme" link.
const readmeURL = "https://github.com/sh-devment/auth-client"

// appName is the app's runtime identity, from APP: the name shown in the UI
// (page title, login card), the --version banner, the start log line and the
// database filename all derive from it.
var appName string

// dbPath is <DB_DIR>/<appName>.db; db.go opens it.
var dbPath string

//go:embed web
var webFiles embed.FS

var (
	authURL      string
	authInternal string
	appURL       string
	appToken     string
	tmpl         *template.Template
	httpClient   = &http.Client{}
)

type pageData struct {
	App        string // appName — shown in <title> and on the login card
	ReadmeURL  string
	User       *User
	Error      string
	Version    string // buildTime — cache-bust token for static assets (?v=)
	*regionDef        // Domain, Lang, Text… — see region.go
}

func initTemplate() {
	src, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		log.Fatalf("web/index.html not found: %v", err)
	}
	tmpl = template.Must(template.New("index").Parse(string(src)))
}

// ── request logging ───────────────────────────────────────────────────────────

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(status int) {
	sw.status = status
	sw.ResponseWriter.WriteHeader(status)
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}

// cacheStatic wraps a handler with a 30-day immutable cache header.
func cacheStatic(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=2592000, immutable") // 30 days; URL is busted per-deploy via ?v=
		h.ServeHTTP(w, r)
	})
}

// ── app routes ────────────────────────────────────────────────────────────────
// Add your app-specific handlers here.

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if code := r.URL.Query().Get("code"); code != "" {
		handleCallback(w, r, code)
		return
	}
	var user *User
	if uid := sessionUserID(r); uid != 0 {
		user, _ = getUserByID(uid)
	}
	// HTML must always revalidate so the fresh ?v= asset token reaches the browser.
	w.Header().Set("Cache-Control", "no-cache")
	tmpl.Execute(w, pageData{ //nolint:errcheck
		App:       appName,
		ReadmeURL: readmeURL,
		User:      user,
		Version:   buildTime,
		regionDef: region,
	})
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
	godotenv.Load() //nolint:errcheck

	// Resolved before anything else: --version prints it, and the database is
	// named after it. Everything downstream reads appName, never a literal.
	appName = os.Getenv("APP")
	if appName == "" {
		appName = defaultAppName
	}

	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "--info") {
		fmt.Printf("%s built: %s\n", appName, buildTime)
		os.Exit(0)
	}

	// The filename always follows the app, so a config copied from another app
	// cannot leave this one writing into that app's database. Only the directory
	// is configurable: dev-compose points it at the /data volume (modernc sqlite
	// locking fails on the macOS bind mount); in prod it is unset and the DB is
	// /opt/$APP/$APP.db — the path deploy.sh and backup.sh expect.
	dbPath = filepath.Join(os.Getenv("DB_DIR"), appName+".db")

	authURL = os.Getenv("AUTH_URL")
	authInternal = os.Getenv("AUTH_INTERNAL")
	appURL = os.Getenv("APP_URL")
	appToken = os.Getenv("APP_TOKEN")
	initRegion()

	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		secretKey = "dev-secret"
	}
	jwtSecret = []byte(secretKey)

	initDB()
	initTemplate()

	webFS, _ := fs.Sub(webFiles, "web")
	fileServer := http.FileServer(http.FS(webFS))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", handleIndex)
	mux.HandleFunc("GET /login", handleLogin)
	mux.HandleFunc("GET /logout", handleLogout)
	mux.HandleFunc("GET /apps", handleOpenApps)
	mux.Handle("GET /favicon.svg", cacheStatic(fileServer))
	mux.Handle("GET /apple-touch-icon.png", cacheStatic(fileServer))
	mux.Handle("GET /style.css", cacheStatic(fileServer))
	mux.Handle("GET /script.js", cacheStatic(fileServer))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8890"
	}
	log.Printf("start app=%s db=%s port=%s", appName, dbPath, port)
	log.Fatal(http.ListenAndServe(":"+port, logMiddleware(mux)))
}
