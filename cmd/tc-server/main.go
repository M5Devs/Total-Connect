package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/M5Devs/Total-Connect/internal/core"
	"github.com/M5Devs/Total-Connect/internal/server"
)

func main() {
	defaultPort := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		defaultPort = envPort
	}

	hostFlag := flag.String("host", "0.0.0.0", "Server listen host")
	portFlag := flag.String("port", defaultPort, "Server listen port")
	configFlag := flag.String("config", "", "Path to custom rclone config file")
	authFlag := flag.String("auth", os.Getenv("TC_AUTH"), "Web authentication credentials in format username:password")
	flag.Parse()

	engine, err := core.NewRcloneEngine(*configFlag)
	if err != nil {
		log.Fatalf("Failed to initialize storage engine: %v", err)
	}

	srv := server.NewServer(engine, server.WithAuth(*authFlag))
	addr := fmt.Sprintf("%s:%s", *hostFlag, *portFlag)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Println("==================================================")
		log.Println("  ⚡ TOTAL CONNECT WEB DAEMON (tc-server)")
		log.Printf("  Total Connect Web Server running at http://%s:%s\n", *hostFlag, *portFlag)
		log.Println("==================================================")

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-stop
	log.Println("\nShutting down Total Connect Web Server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Total Connect Web Server stopped gracefully.")
}
