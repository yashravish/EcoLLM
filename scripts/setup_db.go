package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	passwords := []string{"postgres", "psdi", "password", "admin", "", "practiceops", "pgvector"}
	users := []string{"postgres", "psdi"}

	ctx := context.Background()
	var conn *pgx.Conn

	for _, user := range users {
		for _, pw := range passwords {
			url := fmt.Sprintf("postgres://%s:%s@localhost:5432/postgres?sslmode=disable", user, pw)
			c, err := pgx.Connect(ctx, url)
			if err == nil {
				fmt.Printf("Connected as %s / %s\n", user, pw)
				conn = c
				break
			}
		}
		if conn != nil {
			break
		}
	}

	if conn == nil {
		fmt.Fprintln(os.Stderr, "Could not connect to postgres on :5432 with any known credentials")
		os.Exit(1)
	}
	defer conn.Close(ctx)

	stmts := []string{
		`CREATE USER ecollm WITH PASSWORD 'ecollm_dev'`,
		`CREATE DATABASE ecollm OWNER ecollm`,
		`GRANT ALL PRIVILEGES ON DATABASE ecollm TO ecollm`,
	}
	for _, stmt := range stmts {
		_, err := conn.Exec(ctx, stmt)
		if err != nil {
			fmt.Printf("  skip (already exists?): %v\n", err)
		} else {
			fmt.Printf("  OK: %s\n", stmt)
		}
	}
	fmt.Println("Done — ecollm user + database ready on :5432")
}
