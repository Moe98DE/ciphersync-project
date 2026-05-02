package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/ciphersync/core/postgres"
	"github.com/jackc/pgx/v5"
)

func main() {
	// Provide a local DB connection string (modify if your local DB requires a password)
	connString := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	if envConn := os.Getenv("DATABASE_URL"); envConn != "" {
		connString = envConn
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx)

	introspector := postgres.NewIntrospector(conn)
	tables, err := introspector.Introspect(ctx)
	if err != nil {
		log.Fatalf("Introspection failed: %v", err)
	}

	// Print formatted JSON for Team Alpha to inspect
	out, _ := json.MarshalIndent(tables, "", "  ")
	fmt.Println(string(out))
}
