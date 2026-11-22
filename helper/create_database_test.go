package helper

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"testing"

	"github.com/lib/pq"
)

// pgServer returns the PG_DB_* server (dev stack defaults), failing in CI and skipping locally when it is down.
func pgServer(t *testing.T) (host, port, user, password string) {
	t.Helper()
	get := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}
	host, port = get("PG_DB_HOST", "127.0.0.1"), get("PG_DB_PORT", "5432")
	user, password = get("PG_DB_USER", "postgres"), get("PG_DB_PASSWORD", "!ChangeMe!")
	db, err := sql.Open("postgres", BuildConnInfo(user, password, host, port, ""))
	if err == nil {
		defer func() { _ = db.Close() }()
		err = db.Ping()
	}
	if err != nil {
		if ci := os.Getenv("CI"); ci != "" && ci != "false" && ci != "0" {
			t.Fatalf("Postgres is unreachable and CI is set: %v", err)
		}
		t.Skipf("skipping: Postgres is unreachable: %v", err)
	}
	return host, port, user, password
}

// mustDrop drops the database without FORCE, so it fails while any connection to it is left open.
func mustDrop(t *testing.T, conninfo, name string) {
	t.Helper()
	db, err := sql.Open("postgres", conninfo)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("DROP DATABASE " + pq.QuoteIdentifier(name)); err != nil {
		t.Fatalf("drop %s: %v", name, err)
	}
}

func TestCreateAndRecreateDatabaseLeaveNoConnectionOpen(t *testing.T) {
	host, port, user, password := pgServer(t)
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	name := "dbhelper_" + hex.EncodeToString(b)
	server := BuildConnInfo(user, password, host, port, "")
	t.Cleanup(func() {
		if db, err := sql.Open("postgres", server); err == nil {
			_, _ = db.Exec("DROP DATABASE IF EXISTS " + pq.QuoteIdentifier(name) + " WITH (FORCE)")
			_ = db.Close()
		}
	})

	if err := CreatePostgresDatabase(name, host, port, user, password); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := RecreatePostgresDatabase(name, host, port, user, password); err != nil {
		t.Fatalf("recreate: %v", err)
	}

	db, err := sql.Open("postgres", BuildConnInfo(user, password, host, port, name))
	if err != nil {
		t.Fatal(err)
	}
	var id string
	if err := db.QueryRow("SELECT uuid_generate_v4()::text").Scan(&id); err != nil {
		t.Fatalf("uuid-ossp is not installed in the recreated database: %v", err)
	}
	_ = db.Close()

	mustDrop(t, server, name)
}
