package api

import (
	"net/http"

	"vizhi/backend/internal/auth"
	"vizhi/backend/internal/monitor"
	"vizhi/backend/internal/process"
	"vizhi/backend/internal/transfer"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(
	authn *auth.Authenticator,
	mon *monitor.Monitor,
	appMgr *process.AppManager,
	tm *transfer.TransferManager,
	emitInterval int,
) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Auth middleware (applied to all except /auth/login, /health)
	r.Use(authn.Middleware)

	// Handlers
	statsH := NewStatsHandler(mon)
	filesH := NewFilesHandler(tm)
	appsH := NewAppsHandler(appMgr)
	wsH := NewWSHandler(mon, appMgr, emitInterval)

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/stats", statsH.GetStats)
		r.Get("/stats/stream", wsH.ServeWS)

		r.Get("/files", filesH.List)
		r.Post("/files/upload/init", filesH.InitUpload)
		r.Post("/files/upload/chunk", filesH.UploadChunk)
		r.Post("/files/upload/complete", filesH.FinalizeUpload)
		r.Get("/files/download/{path:.*}", filesH.Download)
		r.Delete("/files/{path:.*}", filesH.Delete)

		r.Get("/apps", appsH.ListApps)
		r.Post("/apps/launch", appsH.Launch)
		r.Post("/apps/terminate", appsH.Terminate)
	})

	// Auth routes (bypass JWT middleware via path check in auth middleware)
	r.Post("/auth/login", authn.LoginHandler)

	// Health
	r.Get("/health", healthCheck)

	return r
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","service":"vizhi"}`))
}
