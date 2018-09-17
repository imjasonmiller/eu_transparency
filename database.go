package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

type postgres struct {
	db                     *sql.DB
	user, pass, host, port string
}

func databaseConn(cfg map[string]string) (postgres, error) {
	connStr := fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=eu_transparency sslmode=disable",
		cfg["DB_USER"], cfg["DB_PASS"], cfg["DB_HOST"], cfg["DB_PORT"],
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return postgres{}, fmt.Errorf("could not open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return postgres{}, fmt.Errorf("could not ping datbase: %v", err)
	}

	return postgres{
		db,
		cfg["DB_USER"],
		cfg["DB_PASS"],
		cfg["DB_HOST"],
		cfg["DB_PORT"],
	}, nil
}

func (p *postgres) Backup() error {
	// If a password is present, add a delimiter.
	var delimit string
	if p.pass != "" {
		delimit = ":"
	}

	// Passing a connection string avoids the password prompt.
	connStr := fmt.Sprintf(
		`postgres://%s%s%s@%s:%s/eu_transparency?sslmode=disable`,
		p.user, delimit, p.pass, p.host, p.port,
	)

	name := fmt.Sprintf("DB_%s.dump", time.Now().Format("2006-01-02"))
	path := filepath.Join("database", "backups", name)

	// See https://www.postgresql.org/docs/10/static/app-pgdump.html for commands.
	cmd := exec.Command("pg_dump", "--dbname", connStr, "-Z", "9", "-F", "c", "-f", path)

	// Capture the more descriptive error and send to stderr.
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not run pg_dump: %s", stderr.String())
	}

	return nil
}

func (p *postgres) Close() {
	p.db.Close()
}
