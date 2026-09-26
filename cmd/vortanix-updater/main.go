package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vortanixapp/panel/internal/updater"
	"github.com/vortanixapp/panel/pkg/netaddr"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "apply":
			os.Exit(updater.Apply(os.Args[2:]))
		case "transfer-import":
			os.Exit(updater.TransferImport())
		}
	}

	secret := os.Getenv("INTERNAL_SECRET")
	if secret == "" {
		log.Fatal("INTERNAL_SECRET is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	upd := updater.NewServer(secret)
	srv := &http.Server{
		Addr:              netaddr.Listen(port),
		Handler:           upd.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("updater listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("updater: %v", err)
		}
	}()

	cleanupCtx, stopCleanup := context.WithCancel(context.Background())
	go upd.CleanupImages(cleanupCtx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	stopCleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
