package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vizhi/backend/internal/api"
	"vizhi/backend/internal/auth"
	"vizhi/backend/internal/config"
	"vizhi/backend/internal/monitor"
	"vizhi/backend/internal/process"
	"vizhi/backend/internal/transfer"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("vizhi-server starting...")

	cfg := config.Load()

	// Master password: either from env (bcrypt hash) or plain password to hash.
	// Pre-compute the hash offline and pass VIZHI_MASTER_PASSWORD_HASH.
	masterHash := os.Getenv("VIZHI_MASTER_PASSWORD_HASH")
	if masterHash == "" {
		plain := os.Getenv("VIZHI_MASTER_PASSWORD")
		if plain == "" {
			log.Fatal("either VIZHI_MASTER_PASSWORD_HASH or VIZHI_MASTER_PASSWORD must be set")
		}
		a := auth.New(cfg.JWTSecret, cfg.JWTIssuer, cfg.TokenDuration, "")
		hash, err := a.SetMasterPassword(plain)
		if err != nil {
			log.Fatalf("hash master password: %v", err)
		}
		masterHash = hash
	}
	// Zero out env after reading to reduce exposure surface
	os.Unsetenv("VIZHI_MASTER_PASSWORD")

	authenticator := auth.New(cfg.JWTSecret, cfg.JWTIssuer, cfg.TokenDuration, masterHash)

	mon := monitor.New(10) // top 10 processes

	appMgr := process.NewAppManager(cfg.AllowedApps, cfg.AllowedUID, cfg.AllowedGID, 30*time.Second)

	tm := transfer.NewTransferManager(cfg.UploadDir, cfg.MaxUploadSize, cfg.ChunkSize)

	router := api.NewRouter(authenticator, mon, appMgr, tm, int(cfg.WSEmitInterval.Seconds()))

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		srv.Close()
	}()

	if cfg.TLSEnabled {
		log.Printf("listening on %s (TLS)", addr)
		if err := srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil && err != http.ErrServerClosed {
			log.Fatalf("tls server: %v", err)
		}
	} else {
		log.Printf("listening on %s (plaintext)", addr)
		log.Println("WARNING: TLS disabled — do not expose to the public internet")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}

	log.Println("server stopped")
}
