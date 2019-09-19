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

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/lucabrasi83/peppamon_cisco/initializer"
	"github.com/lucabrasi83/peppamon_cisco/logging"
)

// ConnPool represents the Connection Pool instance
// db represents an instance of Postgres connection pool
var ConnPool *pgxpool.Pool
var DBInstance *peppamonMetaDB

const (
	shortQueryTimeout = 10 * time.Second
	//mediumQueryTimeout = 3 * time.Minute
	//longQueryTimeout   = 10 * time.Minute
)

type peppamonMetaDB struct {
	db *pgxpool.Pool
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

	// pgx v4 requires config struct to be generated using ParseConfig method
	poolConfig, errParsePool := pgxpool.ParseConfig("")

	if errParsePool != nil {
		logging.PeppaMonLog("fatal", fmt.Sprintf("failed to parse DB pool config %v", errParsePool))
	}

	// Set Connection Parameters
	poolConfig.MaxConns = 50
	poolConfig.HealthCheckPeriod = 5 * time.Second
	poolConfig.ConnConfig.Host = os.Getenv("PEPPAMON_METADB_HOST")
	poolConfig.ConnConfig.User = os.Getenv("PEPPAMON_METADB_USERNAME")
	poolConfig.ConnConfig.Password = os.Getenv("PEPPAMON_METADB_PASSWORD")
	poolConfig.ConnConfig.Database = os.Getenv("PEPPAMON_METADB_DATABASE_NAME")

	poolConfig.ConnConfig.TLSConfig =
		&tls.Config{
			ServerName: os.Getenv("PEPPAMON_METADB_HOST"),
			RootCAs:    certPool,
		}

	poolConfig.ConnConfig.DialFunc =
		(&net.Dialer{
			KeepAlive: 10 * time.Second,
			Timeout:   30 * time.Second,
		}).DialContext

	ConnPool, err = pgxpool.ConnectConfig(context.Background(), poolConfig)

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

func newDBPool(pool *pgxpool.Pool) *peppamonMetaDB {

	return &peppamonMetaDB{
		db: pool,
	}
}

func (p *peppamonMetaDB) displayPostgresVersion() string {
	var version string

	// Set Query timeout
	ctxTimeout, cancelQuery := context.WithTimeout(context.Background(), shortQueryTimeout)

	defer cancelQuery()

	err := p.db.QueryRow(ctxTimeout, "SELECT version()").Scan(&version)

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
