// Command migrate applies and inspects the PostgreSQL schema migrations.
//
// Usage: moada-migrate <up|down|status|version>
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jho-on/moada/internal/database"
)

func main() {
	if len(os.Args) != 2 {
		usage()
		os.Exit(2)
	}

	databaseURL := os.Getenv("MOADA_DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "moada-migrate: MOADA_DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := database.Migrate(ctx, databaseURL, os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "moada-migrate: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: moada-migrate <up|down|status|version>")
}
