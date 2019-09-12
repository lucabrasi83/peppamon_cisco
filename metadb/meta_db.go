// Package metadbdb handles connection and SQL queries to Telemetry Metadata Postgres DB
package metadb

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"sort"
	"time"

	"github.com/jackc/pgx"
	"github.com/lucabrasi83/peppamon_cisco/initializer"
	"github.com/lucabrasi83/peppamon_cisco/logging"
)

// ConnPool represents the Connection Pool instance
// db represents an instance of Postgres connection pool
var ConnPool *pgx.ConnPool
var DBInstance *peppamonMetaDB
var connPoolConfig pgx.ConnPoolConfig

const (
	shortQueryTimeout = 10 * time.Second
	//mediumQueryTimeout = 3 * time.Minute
	//longQueryTimeout   = 10 * time.Minute
)

type peppamonMetaDB struct {
	db *pgx.ConnPool
}

// init() function will establish DB connection pool while package is being loaded.
func init() {

	// Initializer Banner and binary metadata
	initializer.Initialize()

	// Check Environment Variables for Postgres DB Credentials
	if os.Getenv("PEPPAMON_METADB_USERNAME") == "" || os.Getenv("PEPPAMON_METADB_PASSWORD") == "" {
		logging.PeppaMonLog("fatal",
			"Missing Environment Variable(s) for PostgresDB Connection not set ",
			"(PEPPAMON_METADB_USERNAME / PEPPAMON_METADB_PASSWORD)")
	}

	// Check Environment Variables for Postgres Hostname
	if os.Getenv("PEPPAMON_METADB_HOST") == "" {
		logging.PeppaMonLog("fatal",
			"Missing Environment Variable for PostgresDB Hostname ",
			"PEPPAMON_METADB_HOST")
	}

	// Check Environment Variables for Postgres Database Name
	if os.Getenv("PEPPAMON_METADB_DATABASE_NAME") == "" {
		logging.PeppaMonLog("fatal",
			"Missing Environment Variable for PostgresDB Database Name ",
			"PEPPAMON_METADB_DATABASE_NAME")
	}

	// Create a certificate pool from the system certificate authority
	certPool, _ := x509.SystemCertPool()

	var err error

	connPoolConfig = pgx.ConnPoolConfig{
		ConnConfig: pgx.ConnConfig{
			Host: os.Getenv("PEPPAMON_METADB_HOST"),

			TLSConfig: &tls.Config{
				ServerName: os.Getenv("PEPPAMON_METADB_HOST"),
				RootCAs:    certPool,
			},
			User:     os.Getenv("PEPPAMON_METADB_USERNAME"),
			Password: os.Getenv("PEPPAMON_METADB_PASSWORD"),
			Database: os.Getenv("PEPPAMON_METADB_DATABASE_NAME"),
			Dial: (&net.Dialer{
				KeepAlive: 30 * time.Second,
				Timeout:   10 * time.Second,
			}).Dial,
			// TargetSessionAttrs: "read-write",
		},
		MaxConnections: 50,
	}

	ConnPool, err = pgx.NewConnPool(connPoolConfig)

	if err != nil {
		logging.PeppaMonLog(
			"fatal",
			fmt.Sprintf("Unable to Create Postgres Connection Pool: %v", err))
	} else {
		logging.PeppaMonLog("info", "Database Connection Pool successfully created")
	}

	// Instantiate DB object after successful connection
	DBInstance = newDBPool(ConnPool)

	postgresVersion := DBInstance.displayPostgresVersion()

	logging.PeppaMonLog("info", fmt.Sprintf("Postgres SQL Version: %v", postgresVersion))

}

func newDBPool(pool *pgx.ConnPool) *peppamonMetaDB {

	return &peppamonMetaDB{
		db: pool,
	}
}

func (p *peppamonMetaDB) displayPostgresVersion() string {
	var version string

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	defer cancelQuery()

	err := p.db.QueryRowEx(ctxTimeout, "SELECT version()", nil).Scan(&version)

	if err != nil {
		logging.PeppaMonLog(
			"error",
			"Failed to retrieve Postgres Version: ",
			err.Error())
	}

	return version

}

// normalizeString is a helper function that converts empty string to nil pointer.
// Main usage is to convert empty string to Postgres NULL type
func normalizeString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// binarySearchSanitizeDB is a helper function to perform binary searches on integer slices
// Its main usage is during sanitization of data between Telemetry data and DB records
func binarySearchSanitizeDB(indexes []int, item int) bool {

	i := sort.Search(len(indexes), func(i int) bool { return indexes[i] >= item })
	if i < len(indexes) && indexes[i] == item {
		return true
	}
	return false
}
