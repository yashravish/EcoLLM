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

	var userID, orgID string
	err = pool.QueryRow(ctx, "SELECT id, org_id FROM users WHERE email=$1", email).Scan(&userID, &orgID)
	if err != nil {
		fmt.Printf("User %s not found: %v\n", email, err)
		os.Exit(0)
	}
	fmt.Printf("Found: user=%s org=%s\n", userID, orgID)

	// Delete dependents in safe order
	for _, q := range []string{
		"DELETE FROM api_keys WHERE org_id=$1",
		"DELETE FROM organization_members WHERE org_id=$1",
	} {
		if _, e := pool.Exec(ctx, q, orgID); e != nil {
			fmt.Printf("warn (%s): %v\n", q, e)
		}
	}

	if _, err = pool.Exec(ctx, "DELETE FROM users WHERE id=$1", userID); err != nil {
		fmt.Fprintf(os.Stderr, "delete user: %v\n", err)
		os.Exit(1)
	}
	if _, err = pool.Exec(ctx, "DELETE FROM organizations WHERE id=$1", orgID); err != nil {
		fmt.Fprintf(os.Stderr, "delete org: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Done â€” %s can now register fresh.\n", email)
}



