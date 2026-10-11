package repository

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
)

type Config struct {
	User, Password, Host, Port, Name string
	// TLS: "" (off), "true" (system CAs) or "custom" (CACert below).
	TLS    string
	CACert string
}

// ConfigFromEnv reads the DB_* variables, defaulting to a local MySQL.
func ConfigFromEnv() Config {
	return Config{
		User:     getenv("DB_USER", "root"),
		Password: os.Getenv("DB_PASSWORD"),
		Host:     getenv("DB_HOST", "127.0.0.1"),
		Port:     getenv("DB_PORT", "3306"),
		Name:     getenv("DB_NAME", "halloween"),
		TLS:      os.Getenv("DB_TLS"),
		CACert:   os.Getenv("DB_CA_CERT"),
	}
}

func (c Config) dsn(dbName string) string {
	cfg := mysql.NewConfig()
	cfg.User = c.User
	cfg.Passwd = c.Password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(c.Host, c.Port)
	cfg.DBName = dbName
	cfg.TLSConfig = c.TLS
	cfg.ParseTime = true
	cfg.ClientFoundRows = true // UPDATE reports matched rows, even if nothing changed
	return cfg.FormatDSN()
}

// Connect waits for MySQL to be up, creates the database and tables if they
// don't exist and returns a connection pool shared by the whole app.
func Connect(c Config) (*sql.DB, error) {
	// Hosted databases (e.g. Aiven) require TLS with their own CA certificate,
	// passed as PEM text in DB_CA_CERT.
	if c.CACert != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(c.CACert)) {
			return nil, errors.New("DB_CA_CERT is not a valid PEM certificate")
		}
		if err := mysql.RegisterTLSConfig("custom", &tls.Config{RootCAs: pool, ServerName: c.Host}); err != nil {
			return nil, err
		}
		c.TLS = "custom"
	}

	server, err := sql.Open("mysql", c.dsn(""))
	if err != nil {
		return nil, err
	}
	defer server.Close()

	if err := waitFor(server); err != nil {
		return nil, err
	}
	if _, err := server.Exec("CREATE DATABASE IF NOT EXISTS `" + c.Name + "`"); err != nil {
		return nil, fmt.Errorf("creating database: %w", err)
	}

	db, err := sql.Open("mysql", c.dsn(c.Name))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			db.Close()
			return nil, fmt.Errorf("creating tables: %w", err)
		}
	}
	if err := addMissingColumns(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("adding columns: %w", err)
	}
	log.Printf("connected to database %s", c.Name)
	return db, nil
}

// waitFor retries the ping for up to a minute, since MySQL usually takes a
// while to boot when started with docker compose.
func waitFor(db *sql.DB) error {
	const attempts = 30
	var err error
	for i := 1; i <= attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err = db.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}
		log.Printf("waiting for database (%d/%d): %v", i, attempts, err)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("database not reachable: %w", err)
}

var schema = []string{
	`CREATE TABLE IF NOT EXISTS users(
		id int primary key auto_increment,
		is_admin boolean not null,
		name text not null,
		pw_code text not null,
		costume text not null,
		pending_votes int not null,
		created_at datetime default CURRENT_TIMESTAMP,
		updated_at datetime default CURRENT_TIMESTAMP)`,
	`CREATE TABLE IF NOT EXISTS votes(
		id int primary key auto_increment,
		voter_passphrase text not null,
		user_costume_id int not null,
		message text not null,
		created_at datetime default CURRENT_TIMESTAMP,
		updated_at datetime default CURRENT_TIMESTAMP)`,
	// Single row (id = 1) with the game state, so it survives server restarts.
	`CREATE TABLE IF NOT EXISTS game_state(
		id int primary key,
		started boolean not null default false,
		finished boolean not null default false)`,
	`INSERT IGNORE INTO game_state(id) VALUES (1)`,
}

// Columns added after the first release. They are created on databases that
// already exist (CREATE TABLE IF NOT EXISTS doesn't add columns).
var addedColumns = []struct{ table, column, definition string }{
	{"votes", "hidden", "boolean not null default false"},
}

func addMissingColumns(db *sql.DB) error {
	for _, c := range addedColumns {
		var n int
		err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`, c.table, c.column).Scan(&n)
		if err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", c.table, c.column, c.definition)); err != nil {
			return err
		}
		log.Printf("added column %s.%s", c.table, c.column)
	}
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
