// +build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	email := "yashravish@gmail.com"
	if len(os.Args) > 1 {
		email = os.Args[1]
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://ecollm:ecollm_dev@localhost:5432/ecollm?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect error: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Find user
	var userID, orgID string
	err = pool.QueryRow(ctx, "SELECT id, org_id FROM users WHERE email=$1", email).Scan(&userID, &orgID)
	if err != nil {
		fmt.Printf("User %s not found (or query error: %v)\n", email, err)
		os.Exit(0)
	}
	fmt.Printf("Found user %s (id=%s, org=%s)\n", email, userID, orgID)

	tables := []struct{ q string }{
		{"DELETE FROM api_keys WHERE org_id=$1"},
		{"DELETE FROM audit_log WHERE org_id=$1"},
		{"DELETE FROM inference_requests WHERE org_id=$1"},
		{"DELETE FROM usage_daily WHERE org_id=$1"},
		{"DELETE FROM billing_events WHERE org_id=$1"},
		{"DELETE FROM energy_measurements WHERE org_id IS NOT NULL AND org_id=$1"},
		{"DELETE FROM carbon_measurements WHERE org_id IS NOT NULL AND org_id=$1"},
	}
	for _, t := range tables {
		pool.Exec(ctx, t.q, orgID) // best-effort, ignore missing tables
	}

	_, err = pool.Exec(ctx, "DELETE FROM organization_members WHERE user_id=$1", userID)
	if err != nil { fmt.Printf("warn: %v\n", err) }
	_, err = pool.Exec(ctx, "DELETE FROM users WHERE id=$1", userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "delete user error: %v\n", err)
		os.Exit(1)
	}
	_, err = pool.Exec(ctx, "DELETE FROM organizations WHERE id=$1", orgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "delete org error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Deleted user %s and org %s — you can now register fresh.\n", email, orgID)
}
