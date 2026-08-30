package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	pathpkg "path"
	"strings"
	"time"

	"rainhush/internal/config"
)

func Serve(ctx context.Context) error {
	port := fmt.Sprintf("%d", config.Cfg.Server.Port)

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("public"))
	mux.Handle("/", cacheMiddleware(fs))

	srv := &http.Server{
		Addr:           "127.0.0.1:" + port,
		Handler:        securityHeaders(mux),
		ReadTimeout:    15 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		MaxHeaderBytes:  1 << 20,
	}

	shutdownCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	go func() {
		<-shutdownCtx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	fmt.Printf("Serving at http://localhost:%s\n", port)
	fmt.Println("Press Ctrl+C to stop")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	fmt.Println("\nServer stopped.")
	return nil
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

func cacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", cacheControlForPath(r.URL.Path))
		next.ServeHTTP(w, r)
	})
}

func cacheControlForPath(requestPath string) string {
	cleanPath := pathpkg.Clean("/" + strings.TrimSpace(requestPath))
	ext := strings.ToLower(pathpkg.Ext(cleanPath))

	switch {
	case strings.HasPrefix(cleanPath, "/static/bundle.") && (ext == ".css" || ext == ".js"):
		return "public, max-age=31536000, immutable"
	case strings.HasPrefix(cleanPath, "/static/deep.") && ext == ".js":
		return "public, max-age=31536000, immutable"
	case cleanPath == "/" || strings.HasSuffix(requestPath, "/") || ext == ".html" || ext == ".xml" || ext == ".txt" || ext == ".json":
		return "public, max-age=0, must-revalidate"
	default:
		return "public, max-age=2592000"
	}
}
