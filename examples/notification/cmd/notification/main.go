package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"notification-service/internal/handler"
	notificationschema "notification-service/schema"
)

const (
	defaultListenAddr = ":8080"
	defaultDSN        = "postgres://notification:notification@localhost:5432/notification?sslmode=disable"
	defaultServerURL  = "http://localhost:8080"
)

var (
	ErrValidation = errors.New("validation error")
	ErrRequest    = errors.New("request error")
)

func main() {
	if len(os.Args) <= 1 {
		if err := runServe(os.Args[1:]); err != nil {
			log.Fatal(err)
		}

		return
	}

	subcommand := os.Args[1]
	switch subcommand {
	case "serve":
		if err := runServe(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "create":
		if err := runCreate(os.Args[2:]); err != nil {
			log.Fatal(err)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		if strings.HasPrefix(subcommand, "-") {
			if err := runServe(os.Args[1:]); err != nil {
				log.Fatal(err)
			}

			return
		}

		fmt.Fprintf(os.Stderr, "unknown command: %s\n", subcommand)
		printUsage()
		os.Exit(2)
	}
}

func runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	listenAddr := fs.String("listen", defaultListenAddr, "HTTP listen address")
	dsn := fs.String("dsn", defaultDSN, "PostgreSQL DSN")
	initSchema := fs.Bool("init", false, "initialize database schema before serving")
	enableCORS := fs.Bool("cors", true, "enable permissive CORS headers")
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return err
	}

	db, err := sql.Open("pgx", *dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := pingDatabase(db); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if *initSchema {
		if err := applySchema(ctx, db); err != nil {
			return fmt.Errorf("apply schema: %w", err)
		}

		log.Print("schema ensured")
	}

	api := handler.New(db)
	mux := http.NewServeMux()
	api.Register(mux)
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		io.WriteString(w, `{"status":"ok","message":"notification API running","api_prefix":"`+handler.APIPrefix+`"}`)
	})

	var handler = accessLogMiddleware(mux)
	if *enableCORS {
		handler = corsMiddleware(handler)
	}

	server := &http.Server{
		Addr:    *listenAddr,
		Handler: handler,
	}

	log.Printf("notification API listening on %s", *listenAddr)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}

func runCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	title := fs.String("title", "", "notification title")
	body := fs.String("body", "", "notification body")
	important := fs.Bool("important", false, "mark notification as important")
	cancelable := fs.Bool("cancelable", false, "set notification as cancelable")
	iconURL := fs.String("icon-url", "", "icon URL (optional)")
	expiresAt := fs.String("expires-at", "", "expiration timestamp (RFC3339, optional)")
	serverURL := fs.String("server", defaultServerURL, "notification server base URL")
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*title) == "" {
		return fmt.Errorf("%w: title is required", ErrValidation)
	}

	if strings.TrimSpace(*body) == "" {
		return fmt.Errorf("%w: body is required", ErrValidation)
	}

	payload := map[string]any{
		"title":      *title,
		"body":       *body,
		"important":  *important,
		"cancelable": *cancelable,
	}

	if *iconURL != "" {
		payload["icon_url"] = *iconURL
	}

	if *expiresAt != "" {
		payload["expires_at"] = *expiresAt
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode payload: %w", err)
	}

	url := strings.TrimRight(*serverURL, "/") + handler.APIPrefix + "/notifications"

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("%w: server returned %s: %s", ErrRequest, resp.Status, strings.TrimSpace(string(bodyBytes)))
	}

	fmt.Println(strings.TrimSpace(string(bodyBytes)))

	return nil
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  notification serve [flags]   Start API server")
	fmt.Println("  notification create [flags]  Create a notification via API")
	fmt.Println()
	fmt.Println("Serve flags:")
	fmt.Println("  -listen string   HTTP listen address (default \"" + defaultListenAddr + "\")")
	fmt.Println("  -dsn string      PostgreSQL DSN (default \"" + defaultDSN + "\")")
	fmt.Println("  -init bool       Initialize database schema before serving")
	fmt.Println("  -cors bool       Enable permissive CORS (default true)")
	fmt.Println()
	fmt.Println("Create flags:")
	fmt.Println("  -title string        Notification title (required)")
	fmt.Println("  -body string         Notification body (required)")
	fmt.Println("  -important           Mark as important")
	fmt.Println("  -cancelable          Mark as cancelable")
	fmt.Println("  -icon-url string     Icon URL")
	fmt.Println("  -expires-at string   Expiration timestamp (RFC3339)")
	fmt.Println("  -server string       Notification server base URL (default \"" + defaultServerURL + "\")")
}

func pingDatabase(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.PingContext(ctx)
}

func applySchema(ctx context.Context, db *sql.DB) error {
	stmts := strings.Split(notificationschema.SchemaSQL, ";")

	for _, raw := range stmts {
		stmt := strings.TrimSpace(raw)
		if stmt == "" {
			continue
		}

		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}

func accessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(lrw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, lrw.status, time.Since(start))
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter

	status int
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	return w.ResponseWriter.Write(b)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
