// Package mcputil provides shared MCP server infrastructure:
// config parsing, graceful shutdown, health check, text results.
package mcputil

import (
	"context"
	"flag"
	"fmt"
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
func Run(cfg ServerConfig, register func(*mcp.Server)) error {
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

	wrapped := withMiddleware(mux)

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

// withMiddleware applies request logging and panic recovery.
func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[%s] PANIC %s %s: %v",
					r.URL.Path, r.Method, r.URL.Path, rec)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()

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
