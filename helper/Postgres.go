package helper

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	config "github.com/CodeClarityCE/utility-types/config_db"
	plugin "github.com/CodeClarityCE/utility-types/plugin_db"
	_ "github.com/lib/pq"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func CreatePostgresDatabase(dbName string, host string, port string, user string, password string) error {
	conninfo := fmt.Sprintf("user=%s password=%s host=%s port=%s sslmode=disable", user, password, host, port)
	pg_connection, err := sql.Open("postgres", conninfo)
	if err != nil {
		log.Fatal(err)
	}

	_, err = pg_connection.Exec("create database " + dbName)
	if err != nil {
		//handle the error
		log.Fatal(err)
	}

	pg_connection.Close()

	conninfo = fmt.Sprintf("user=%s password=%s host=%s port=%s database=%s sslmode=disable", user, password, host, port, dbName)
	pg_connection, err = sql.Open("postgres", conninfo)
	if err != nil {
		log.Fatal(err)
	}
	defer pg_connection.Close()

	_, err = pg_connection.Query("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func RecreatePostgresDatabase(dbName string, host string, port string, user string, password string) error {
	conninfo := fmt.Sprintf("user=%s password=%s host=%s port=%s sslmode=disable", user, password, host, port)
	pg_connection, err := sql.Open("postgres", conninfo)
	if err != nil {
		log.Fatal(err)
	}

	_, err = pg_connection.Exec("drop database " + dbName)
	if err != nil {
		//handle the error
		log.Fatal(err)
	}

	_, err = pg_connection.Exec("create database " + dbName)
	if err != nil {
		//handle the error
		log.Fatal(err)
	}
	pg_connection.Close()

	conninfo = fmt.Sprintf("user=%s password=%s host=%s port=%s database=%s sslmode=disable", user, password, host, port, dbName)
	pg_connection, err = sql.Open("postgres", conninfo)
	if err != nil {
		log.Fatal(err)
	}
	defer pg_connection.Close()

	_, err = pg_connection.Query("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";")
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func CreateDatabase(dbName string, confirm bool) error {
	dbName = strings.ToLower(dbName)
	host := os.Getenv("PG_DB_HOST")
	if host == "" {
		log.Printf("PG_DB_HOST is not set")
		return fmt.Errorf("PG_DB_HOST is not set")
	}
	port := os.Getenv("PG_DB_PORT")
	if port == "" {
		log.Printf("PG_DB_PORT is not set")
		return fmt.Errorf("PG_DB_PORT is not set")
	}
	user := os.Getenv("PG_DB_USER")
	if user == "" {
		log.Printf("PG_DB_USER is not set")
		return fmt.Errorf("PG_DB_USER is not set")
	}
	password := os.Getenv("PG_DB_PASSWORD")
	if password == "" {
		log.Printf("PG_DB_PASSWORD is not set")
		return fmt.Errorf("PG_DB_PASSWORD is not set")
	}

	dsn := "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + dbName + "?sslmode=disable"
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

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
	host := os.Getenv("PG_DB_HOST")
	if host == "" {
		log.Printf("PG_DB_HOST is not set")
		return fmt.Errorf("PG_DB_HOST is not set")
	}
	port := os.Getenv("PG_DB_PORT")
	if port == "" {
		log.Printf("PG_DB_PORT is not set")
		return fmt.Errorf("PG_DB_PORT is not set")
	}
	user := os.Getenv("PG_DB_USER")
	if user == "" {
		log.Printf("PG_DB_USER is not set")
		return fmt.Errorf("PG_DB_USER is not set")
	}
	password := os.Getenv("PG_DB_PASSWORD")
	if password == "" {
		log.Printf("PG_DB_PASSWORD is not set")
		return fmt.Errorf("PG_DB_PASSWORD is not set")
	}

	dsn := "postgres://" + user + ":" + password + "@" + host + ":" + port + "/" + dbName + "?sslmode=disable"
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
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
	db.NewCreateTable().Model((*config.Config)(nil)).IfNotExists().Exec(context.Background())
	db.NewInsert().Model(&config.Config{
		NvdLast: time.Date(2005, 1, 1, 0, 0, 0, 0, time.UTC),
		NpmLast: "0",
	}).Exec(context.Background())
	return nil
}

func createPluginTable(db *bun.DB) error {
	_, err := db.NewCreateTable().Model((*plugin.Plugin)(nil)).IfNotExists().Exec(context.Background())
	if err != nil {
		return err
	}
	return nil
}
