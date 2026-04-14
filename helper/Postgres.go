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
	sslMode := GetSSLMode()
	switch sslMode {
	case "require":
		return pgdriver.WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		})
	case "verify-ca", "verify-full":
		tlsConfig := &tls.Config{}
		if host := os.Getenv("PG_DB_HOST"); host != "" {
			tlsConfig.ServerName = host
		}
		if rootCert := os.Getenv("PG_DB_SSLROOTCERT"); rootCert != "" {
			if caCert, err := os.ReadFile(rootCert); err == nil {
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

func CreatePostgresDatabase(dbName string, host string, port string, user string, password string) error {
	if err := validateDatabaseName(dbName); err != nil {
		return err
	}

	conninfo := BuildConnInfo(user, password, host, port, "")
	pg_connection, err := sql.Open("postgres", conninfo)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	_, err = pg_connection.Exec("CREATE DATABASE " + pq.QuoteIdentifier(dbName))
	if err != nil {
		pg_connection.Close()
		return fmt.Errorf("failed to create database %s: %w", dbName, err)
	}

	pg_connection.Close()

	conninfo = BuildConnInfo(user, password, host, port, dbName)
	pg_connection, err = sql.Open("postgres", conninfo)
	if err != nil {
		return fmt.Errorf("failed to open connection to new database: %w", err)
	}
	defer pg_connection.Close()

	_, err = pg_connection.Query("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	if err != nil {
		return fmt.Errorf("failed to create uuid-ossp extension: %w", err)
	}

	return nil
}

func RecreatePostgresDatabase(dbName string, host string, port string, user string, password string) error {
	if err := validateDatabaseName(dbName); err != nil {
		return err
	}

	conninfo := BuildConnInfo(user, password, host, port, "")
	pg_connection, err := sql.Open("postgres", conninfo)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	_, err = pg_connection.Exec("DROP DATABASE " + pq.QuoteIdentifier(dbName))
	if err != nil {
		pg_connection.Close()
		return fmt.Errorf("failed to drop database %s: %w", dbName, err)
	}

	_, err = pg_connection.Exec("CREATE DATABASE " + pq.QuoteIdentifier(dbName))
	if err != nil {
		pg_connection.Close()
		return fmt.Errorf("failed to create database %s: %w", dbName, err)
	}
	pg_connection.Close()

	conninfo = BuildConnInfo(user, password, host, port, dbName)
	pg_connection, err = sql.Open("postgres", conninfo)
	if err != nil {
		return fmt.Errorf("failed to open connection to new database: %w", err)
	}
	defer pg_connection.Close()

	_, err = pg_connection.Query("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	if err != nil {
		return fmt.Errorf("failed to create uuid-ossp extension: %w", err)
	}
	return nil
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

	dsn := BuildDSN(user, password, host, port, dbName)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn), BuildPgdriverTLSOption()))

	db := bun.NewDB(sqldb, pgdialect.New())
	defer db.Close()

	// If the database doesn't exist, create it
	if err := db.Ping(); err != nil {
		db.Close()
		err = CreatePostgresDatabase(dbName, host, port, user, password)
		return err
	}
	db.Close()

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

	if dbName == Config.Database.Config {
		err := createConfigTable(db)
		if err != nil {
			return err
		}
	} else if dbName == Config.Database.Plugins {
		err := createPluginTable(db)
		if err != nil {
			return err
		}
	} else {
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
