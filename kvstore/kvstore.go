package kvstore

import (
	"net"
	"os"

	"github.com/go-redis/redis/v7"
)

var kvStoreClient = newKVStoreClient()

func newKVStoreClient() *redis.Client {
	// Set Redis Client options
	redisClient := redis.NewClient(&redis.Options{
		Addr:         os.Getenv("PEPPAMON_KV_HOST") + ":6379",
		Password:     os.Getenv("PEPPAMON_KV_PASSWORD"),
		DB:           0, // use default DB,
		PoolSize:     20,
		MinIdleConns: 5,
	})

	return redisClient
}

func InsertNewTelemetryHost(ip net.IPAddr, kv map[string]interface{}) error {

	err := kvStoreClient.HSet(ip.IP.String(), "hostname", kv["hostname"]).Err()
	if err != nil {
		return err
	}

	return nil
}

func LookupTelemetryHost(ip net.IPAddr) (map[string]string, error) {
	res, err := kvStoreClient.HGetAll(ip.IP.String()).Result()

	if err != nil {
		return nil, err
	}
	return res, nil
}
