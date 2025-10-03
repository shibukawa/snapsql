package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver (CGO)

	"github.com/shibukawa/snapsql/examples/kanban"
	"github.com/shibukawa/snapsql/examples/kanban/internal/handler"
)

// simple environment config
var (
	dsn      = flag.String("dsn", "file:kanban.db?cache=shared&mode=rwc", "SQLite DSN")
	migrate  = flag.Bool("migrate", true, "auto create tables if not exist (demo)")
	initFlag = flag.Bool("init", false, "initialize database schema and exit")
)

func main() {
	flag.Parse()

	preparedPath, prepErr := prepareSQLiteFile(*dsn)
	if prepErr != nil && !errors.Is(prepErr, ErrInMemoryDSN) {
		log.Fatalf("prepare sqlite file: %v", prepErr)
	}

	db, err := sql.Open("sqlite3", *dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := pingDatabase(db); err != nil {
		log.Fatalf("ping db: %v", err)
	}

	// Log database file path
	if preparedPath != "" {
		log.Printf("Connected to database: %s", preparedPath)
	} else {
		log.Printf("Connected to database: %s", *dsn)
	}

	if *initFlag {
		if err := initializeSchema(db); err != nil {
			log.Fatalf("apply schema: %v", err)
		}

		if preparedPath != "" {
			log.Printf("database initialized at %s", preparedPath)
		} else {
			log.Printf("database initialized (memory dsn)")
		}

		return
	}

	if *migrate {
		if err := initializeSchema(db); err != nil {
			log.Fatalf("apply schema: %v", err)
		}
	}

	api := handler.New(db)
	mux := http.NewServeMux()
	api.Register(mux)

	configureSPA(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: accessLogMiddleware(mux),
	}
	log.Printf("Backend server starting on %s", server.Addr)

	err = server.ListenAndServe()
	if err != nil {
		log.Printf("server error: %v", err)
	}
}

func pingDatabase(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

func initializeSchema(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return applySchema(ctx, db)
}

func configureSPA(mux *http.ServeMux) {
	// Development: frontend dev server handles SPA routing (no backend SPA needed)
	// Production: use embedded assets only
	if embedded, err := kanban.EmbeddedDistFS(); err == nil {
		if handler, err := newSPAHandlerFromFS(embedded); err == nil {
			log.Printf("serving embedded frontend assets")
			mux.Handle("/", handler)

			return
		} else {
			log.Printf("warning: failed to create SPA handler from embedded assets: %v", err)
		}
	}

	// No embedded assets available - API-only mode (development)
	log.Printf("info: serving API-only mode (no frontend assets)")
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"API server running - use frontend dev server for UI","api_prefix":"/api"}`))
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter

	status int
	body   bytes.Buffer
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	w.body.Write(b)

	return w.ResponseWriter.Write(b)
}

func accessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lrw := &loggingResponseWriter{ResponseWriter: w}

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)

		logMessage := fmt.Sprintf(
			"[%d] %s %s - %v",
			lrw.status,
			r.Method,
			r.URL.Path,
			duration,
		)

		if lrw.status >= 400 {
			logMessage += "\n\tResponse Body: " + lrw.body.String()
		}

		log.Println(logMessage)
	})
}

// applySchema loads and executes schema SQL from embedded sql/schema.sql file.
func applySchema(ctx context.Context, db *sql.DB) error {
	// Split the schema SQL by semicolons to get individual statements
	stmts := strings.Split(kanban.SchemaSQL, ";")

	for _, stmt := range stmts {
		// Trim whitespace and skip empty statements
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}

type spaHandler struct {
	fs         fs.FS
	fileServer http.Handler
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, handler.APIPrefix) {
		http.NotFound(w, r)
		return
	}

	relPath := strings.TrimPrefix(r.URL.Path, "/")
	if relPath == "" {
		h.serveIndex(w, r)
		return
	}

	if !fs.ValidPath(relPath) {
		http.NotFound(w, r)
		return
	}

	info, err := fs.Stat(h.fs, relPath)
	if err == nil {
		if info.IsDir() {
			h.serveIndex(w, r)
			return
		}

		h.fileServer.ServeHTTP(w, r)

		return
	}

	if !errors.Is(err, fs.ErrNotExist) {
		log.Printf("static asset lookup error (%s): %v", relPath, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	h.serveIndex(w, r)
}

func (h *spaHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	file, err := h.fs.Open("index.html")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.Error(w, "frontend index not found", http.StatusServiceUnavailable)
			return
		}

		log.Printf("index lookup error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		log.Printf("index stat error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("index read error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)

		return
	}

	http.ServeContent(w, r, "index.html", info.ModTime(), bytes.NewReader(data))
}
