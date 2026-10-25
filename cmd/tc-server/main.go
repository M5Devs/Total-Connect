package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/M5Devs/Total-Connect/internal/core"
	"github.com/M5Devs/Total-Connect/internal/server"
)

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.IsLoopback() {
		return true
	}
	return false
}

func validateSecurity(host, auth string, insecureNoAuth bool) (warn bool, err error) {
	if !isLoopback(host) && strings.TrimSpace(auth) == "" {
		if !insecureNoAuth {
			return false, fmt.Errorf("SECURITY ERROR: Refusing to bind to non-loopback interface (%s) without authentication. Set -auth username:password, export TC_AUTH, or pass -insecure-no-auth.", host)
		}
		return true, nil
	}
	return false, nil
}

func main() {
	defaultPort := "8080"
	if envPort := os.Getenv("PORT"); envPort != "" {
		defaultPort = envPort
	}

	hostFlag := flag.String("host", "127.0.0.1", "Server listen host")
	portFlag := flag.String("port", defaultPort, "Server listen port")
	configFlag := flag.String("config", "", "Path to custom rclone config file")
	authFlag := flag.String("auth", os.Getenv("TC_AUTH"), "Web authentication credentials in format username:password")
	insecureNoAuthFlag := flag.Bool("insecure-no-auth", false, "Allow binding to non-loopback interfaces without authentication (INSECURE)")
	flag.Parse()

	warn, err := validateSecurity(*hostFlag, *authFlag, *insecureNoAuthFlag)
	if err != nil {
		log.Fatalf("%v", err)
	}

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
		if warn {
			log.Println("==================================================")
			log.Println("  ⚠️  WARNING: INSECURE SERVER CONFIGURATION")
			log.Printf("  Server binding to non-loopback interface (%s)\n", *hostFlag)
			log.Println("  WITHOUT AUTHENTICATION. Storage engine is exposed!")
			log.Println("==================================================")
		}

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
