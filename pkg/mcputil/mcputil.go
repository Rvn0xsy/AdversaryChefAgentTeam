// Package mcputil provides shared MCP server infrastructure:
// config parsing, graceful shutdown, health check, text results.
package mcputil

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerConfig holds common server configuration.
type ServerConfig struct {
	Host         string
	Port         int
	DBPath       string
	Name         string
	Version      string
	MythicServer string
	MythicAPIKey string
}

// Addr returns the listen address.
func (c ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// ParseConfig parses CLI flags and returns a ServerConfig.
func ParseConfig(name, version string, defaultPort int) ServerConfig {
	host := flag.String("host", "0.0.0.0", "host to listen on")
	port := flag.Int("port", defaultPort, "port to listen on")
	dbPath := flag.String("db", "asset.db", "sqlite database file path")
	mythicServer := flag.String("mythic-server", "", "Mythic C2 server URL")
	mythicAPIKey := flag.String("mythic-api-key", "", "Mythic C2 API key")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s MCP Server v%s\n\n", name, version)
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	return ServerConfig{
		Host:         *host,
		Port:         *port,
		DBPath:       *dbPath,
		Name:         name,
		Version:      version,
		MythicServer: *mythicServer,
		MythicAPIKey: *mythicAPIKey,
	}
}

// ---- Context helpers for session-bound project_id ----

type ctxKey int

const CtxKeyProjectID ctxKey = iota

// ProjectIDFromContext extracts the project_id injected by session-binding middleware.
func ProjectIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(CtxKeyProjectID).(string)
	return id
}

// WithProjectID injects a project_id into the context.
func WithProjectID(ctx context.Context, projectID string) context.Context {
	return context.WithValue(ctx, CtxKeyProjectID, projectID)
}

// AddLoggingTool registers a tool handler with logging: prints tool name and params on call.
func AddLoggingTool[P any](server *mcp.Server, tool *mcp.Tool, handler func(context.Context, *mcp.CallToolRequest, P) (*mcp.CallToolResult, any, error)) {
	wrapped := func(ctx context.Context, req *mcp.CallToolRequest, params P) (*mcp.CallToolResult, any, error) {
		start := time.Now()
		log.Printf("[%s] called params=%+v", tool.Name, params)
		result, meta, err := handler(ctx, req, params)
		log.Printf("[%s] done in %v err=%v", tool.Name, time.Since(start), err)
		return result, meta, err
	}
	mcp.AddTool(server, tool, wrapped)
}

// TextResult creates a text content MCP result.
func TextResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// Run starts the MCP server with graceful shutdown.
// sm may be nil for servers that do not use session-to-project binding.
func Run(cfg ServerConfig, register func(*mcp.Server), sm *SessionMap) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    cfg.Name,
		Version: cfg.Version,
	}, nil)

	register(server)

	handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		SessionTimeout: 5 * time.Minute,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler(cfg))
	mux.Handle("/", handler)

	wrapped := withMiddleware(mux, sm)

	srv := &http.Server{
		Addr:    cfg.Addr(),
		Handler: wrapped,
	}

	go func() {
		log.Printf("[%s] listening on %s", cfg.Name, cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[%s] server failed: %v", cfg.Name, err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Printf("[%s] shutting down...", cfg.Name)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func healthHandler(cfg ServerConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","server":"%s","version":"%s"}`, cfg.Name, cfg.Version)
	}
}

// withMiddleware applies request logging, session-binding enforcement, and panic recovery.
func withMiddleware(next http.Handler, sm *SessionMap) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[%s] PANIC %s %s: %v",
					r.URL.Path, r.Method, r.URL.Path, rec)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

		// ---- Session-binding enforcement (only when SessionMap is provided) ----
		if sm != nil && r.Method == http.MethodPost {
			sessionID := r.Header.Get("Mcp-Session-Id")
			if sessionID != "" {
				body, readErr := io.ReadAll(r.Body)
				if readErr == nil {
					r.Body = io.NopCloser(bytes.NewReader(body))

					var toolCall struct {
						Params struct {
							ProjectID string `json:"project_id"`
						} `json:"params"`
					}
					if json.Unmarshal(body, &toolCall) == nil && toolCall.Params.ProjectID != "" {
						projectID, bindErr := sm.GetOrBind(sessionID, toolCall.Params.ProjectID)
						if bindErr != nil {
							log.Printf("[session] binding conflict session=%s project=%s: %v",
								sessionID, toolCall.Params.ProjectID, bindErr)
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusForbidden)
							fmt.Fprintf(w, `{"error":"project binding conflict: %s"}`, bindErr.Error())
							return
						}
						ctx := WithProjectID(r.Context(), projectID)
						r = r.WithContext(ctx)
						log.Printf("[session] bound session=%s -> project=%s", sessionID, projectID)
					}
				}
			}
		}

		next.ServeHTTP(w, r)

		log.Printf("[%s] %s %s %v",
			r.URL.Path, r.Method, r.URL.Path, time.Since(start))
	})
}

// SessionBinding holds the project_id bound to an MCP session.
type SessionBinding struct {
	ProjectID string
	Bound     bool
	BoundAt   time.Time
}

// SessionMap provides concurrent-safe session-to-project binding.
type SessionMap struct {
	mu       sync.RWMutex
	sessions map[string]*SessionBinding
}

// NewSessionMap creates a new empty SessionMap.
func NewSessionMap() *SessionMap {
	return &SessionMap{sessions: make(map[string]*SessionBinding)}
}

// GetOrBind retrieves the project_id for a session, binding it if first seen.
// Returns an error if a different project_id attempts to reuse the same session.
func (m *SessionMap) GetOrBind(sessionID, callerProjectID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, exists := m.sessions[sessionID]
	if !exists {
		m.sessions[sessionID] = &SessionBinding{ProjectID: callerProjectID, Bound: true, BoundAt: time.Now()}
		return callerProjectID, nil
	}
	if s.ProjectID != callerProjectID {
		return "", fmt.Errorf("session %s bound to project %s, rejected %s", sessionID, s.ProjectID, callerProjectID)
	}
	return s.ProjectID, nil
}
