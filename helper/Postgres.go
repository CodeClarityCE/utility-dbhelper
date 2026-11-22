package helper

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"

	config "github.com/CodeClarityCE/utility-types/config_db"
	plugin "github.com/CodeClarityCE/utility-types/plugin_db"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

var validDBName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// validateDatabaseName checks that the database name is safe for use in DDL statements.
func validateDatabaseName(name string) error {
	if !validDBName.MatchString(name) {
		return fmt.Errorf("invalid database name: %q", name)
	}
	return nil
}

// GetSSLMode returns the configured SSL mode from PG_DB_SSLMODE, defaulting based on ENV.
func GetSSLMode() string {
	sslMode := os.Getenv("PG_DB_SSLMODE")
	if sslMode != "" {
		return sslMode
	}
	env := os.Getenv("ENV")
	if env == "prod" || env == "production" {
		return "require"
	}
	return "disable"
}

// BuildConnInfo builds a key=value connection string for lib/pq with SSL support.
func BuildConnInfo(user, password, host, port, dbName string) string {
	sslMode := GetSSLMode()
	var connInfo string
	if dbName == "" {
		connInfo = fmt.Sprintf("user=%s password=%s host=%s port=%s sslmode=%s",
			user, password, host, port, sslMode)
	} else {
		connInfo = fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s",
			user, password, host, port, dbName, sslMode)
	}
	if rootCert := os.Getenv("PG_DB_SSLROOTCERT"); rootCert != "" {
		connInfo += " sslrootcert=" + rootCert
	}
	if cert := os.Getenv("PG_DB_SSLCERT"); cert != "" {
		connInfo += " sslcert=" + cert
	}
	if key := os.Getenv("PG_DB_SSLKEY"); key != "" {
		connInfo += " sslkey=" + key
	}
	return connInfo
}

// BuildDSN builds a postgres:// URI with SSL support.
// User and password are URL-encoded to handle special characters safely.
func BuildDSN(user, password, host, port, dbName string) string {
	sslMode := GetSSLMode()
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		url.QueryEscape(user), url.QueryEscape(password), host, port, dbName, sslMode)

	if rootCert := os.Getenv("PG_DB_SSLROOTCERT"); rootCert != "" {
		dsn += "&sslrootcert=" + rootCert
	}
	if cert := os.Getenv("PG_DB_SSLCERT"); cert != "" {
		dsn += "&sslcert=" + cert
	}
	if key := os.Getenv("PG_DB_SSLKEY"); key != "" {
		dsn += "&sslkey=" + key
	}
	return dsn
}

// BuildPgdriverTLSOption returns a pgdriver.Option that explicitly configures TLS
// based on PG_DB_SSLMODE. This bypasses pgdriver's DSN-based SSL parsing, which
// can silently fall back to non-TLS on cert loading errors.
func BuildPgdriverTLSOption() pgdriver.Option {
	return buildPgdriverTLSOption(logrus.StandardLogger())
}

// buildPgdriverTLSOption is BuildPgdriverTLSOption with the logger passed in.
func buildPgdriverTLSOption(log logrus.FieldLogger) pgdriver.Option {
	sslMode := GetSSLMode()
	switch sslMode {
	case "require":
		// require encrypts without checking the certificate, as in libpq.
		log.WithFields(logrus.Fields{"component": "postgres", "ssl_mode": "require"}).
			Warn("Postgres TLS does not verify the server certificate in require mode; use verify-ca")
		return pgdriver.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // libpq's require mode: encrypt without verifying, warned above
		})
	case "verify-ca", "verify-full":
		tlsConfig := &tls.Config{}
		if host := os.Getenv("PG_DB_HOST"); host != "" {
			tlsConfig.ServerName = host
		}
		if rootCert := os.Getenv("PG_DB_SSLROOTCERT"); rootCert != "" {
			if caCert, err := os.ReadFile(rootCert); err == nil { //nolint:gosec // CA path set by the operator, like libpq's sslrootcert
				pool := x509.NewCertPool()
				pool.AppendCertsFromPEM(caCert)
				tlsConfig.RootCAs = pool
			}
		}
		return pgdriver.WithTLSConfig(tlsConfig)
	default:
		return pgdriver.WithInsecure(true)
	}
}

// getAdminCredentials returns admin-level DB credentials for DDL operations.
// Falls back to PG_DB_USER/PG_DB_PASSWORD if admin-specific vars are not set.
func getAdminCredentials() (string, string) {
	user := os.Getenv("PG_DB_ADMIN_USER")
	if user == "" {
		user = os.Getenv("PG_DB_USER")
	}
	password := os.Getenv("PG_DB_ADMIN_PASSWORD")
	if password == "" {
		password = os.Getenv("PG_DB_PASSWORD")
	}
	return user, password
}

// withConn opens conninfo, runs fn and closes the connection, returning the first error.
func withConn(conninfo string, fn func(*sql.DB) error) (err error) {
	conn, err := sql.Open("postgres", conninfo)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("failed to close connection: %w", closeErr)
		}
	}()
	return fn(conn)
}

// installUUIDExtension adds uuid-ossp to the database dbName.
func installUUIDExtension(dbName string, host string, port string, user string, password string) error {
	return withConn(BuildConnInfo(user, password, host, port, dbName), func(conn *sql.DB) error {
		if _, err := conn.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"); err != nil {
			return fmt.Errorf("failed to create uuid-ossp extension: %w", err)
		}
		return nil
	})
}

func CreatePostgresDatabase(dbName string, host string, port string, user string, password string) error {
	if err := validateDatabaseName(dbName); err != nil {
		return err
	}

	err := withConn(BuildConnInfo(user, password, host, port, ""), func(conn *sql.DB) error {
		if _, err := conn.Exec("CREATE DATABASE " + pq.QuoteIdentifier(dbName)); err != nil {
			return fmt.Errorf("failed to create database %s: %w", dbName, err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return installUUIDExtension(dbName, host, port, user, password)
}

func RecreatePostgresDatabase(dbName string, host string, port string, user string, password string) error {
	if err := validateDatabaseName(dbName); err != nil {
		return err
	}

	err := withConn(BuildConnInfo(user, password, host, port, ""), func(conn *sql.DB) error {
		if _, err := conn.Exec("DROP DATABASE " + pq.QuoteIdentifier(dbName)); err != nil {
			return fmt.Errorf("failed to drop database %s: %w", dbName, err)
		}
		if _, err := conn.Exec("CREATE DATABASE " + pq.QuoteIdentifier(dbName)); err != nil {
			return fmt.Errorf("failed to create database %s: %w", dbName, err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return installUUIDExtension(dbName, host, port, user, password)
}

func CreateDatabase(dbName string, confirm bool) error {
	dbName = strings.ToLower(dbName)
	if err := validateDatabaseName(dbName); err != nil {
		return err
	}

	host := os.Getenv("PG_DB_HOST")
	if host == "" {
		return fmt.Errorf("PG_DB_HOST is not set")
	}
	port := os.Getenv("PG_DB_PORT")
	if port == "" {
		return fmt.Errorf("PG_DB_PORT is not set")
	}

	user, password := getAdminCredentials()
	if user == "" {
		return fmt.Errorf("PG_DB_USER (or PG_DB_ADMIN_USER) is not set")
	}
	if password == "" {
		return fmt.Errorf("PG_DB_PASSWORD (or PG_DB_ADMIN_PASSWORD) is not set")
	}

	// If the database doesn't exist, create it
	if !databaseExists(BuildDSN(user, password, host, port, dbName)) {
		return CreatePostgresDatabase(dbName, host, port, user, password)
	}

	confirmation := "y"
	// If confirmation asked
	if confirm {
		log.Printf("Database %s already exists", dbName)
		log.Printf("Do you want to delete and recreate the database %s? (y/n)", dbName)
		_, err := fmt.Scanln(&confirmation)
		if err != nil {
			return err
		}
	}

	// If confirmation is not y, return
	if confirmation != "y" {
		return nil
	}

	// If confirmation is y, delete the database and create it again
	log.Printf("Deleting database %s", dbName)

	return RecreatePostgresDatabase(dbName, host, port, user, password)
}

// databaseExists reports whether a connection to dsn succeeds; the connection is closed before returning.
func databaseExists(dsn string) bool {
	db := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn), BuildPgdriverTLSOption()))
	defer db.Close()
	return db.Ping() == nil
}

func CreateTable(dbName string) error {
	dbName = strings.ToLower(dbName)
	if err := validateDatabaseName(dbName); err != nil {
		return err
	}

	host := os.Getenv("PG_DB_HOST")
	if host == "" {
		return fmt.Errorf("PG_DB_HOST is not set")
	}
	port := os.Getenv("PG_DB_PORT")
	if port == "" {
		return fmt.Errorf("PG_DB_PORT is not set")
	}

	user, password := getAdminCredentials()
	if user == "" {
		return fmt.Errorf("PG_DB_USER (or PG_DB_ADMIN_USER) is not set")
	}
	if password == "" {
		return fmt.Errorf("PG_DB_PASSWORD (or PG_DB_ADMIN_PASSWORD) is not set")
	}

	dsn := BuildDSN(user, password, host, port, dbName)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn), BuildPgdriverTLSOption()))
	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	switch dbName {
	case Config.Database.Config:
		if err := createConfigTable(db); err != nil {
			return err
		}
	case Config.Database.Plugins:
		if err := createPluginTable(db); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown database name %s", dbName)
	}

	return nil
}

func createConfigTable(db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*config.Config)(nil)).IfNotExists().Exec(context.Background())
	return err
}

func createPluginTable(db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*plugin.Plugin)(nil)).IfNotExists().Exec(context.Background())
	return err
}
