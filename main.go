package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	grpcRecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/lucabrasi83/peppamon_cisco/kvstore"
	"github.com/lucabrasi83/peppamon_cisco/logging"
	"github.com/lucabrasi83/peppamon_cisco/metadb"
	"github.com/lucabrasi83/peppamon_cisco/metrics"
	"github.com/lucabrasi83/peppamon_cisco/proto/mdt_grpc_dialout"
	"github.com/lucabrasi83/peppamon_cisco/proto/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type HighObsSrv struct {
	exp *metrics.Collector
	mu  *sync.Mutex
}

var (
	// Register Peppamon Prometheus Collector
	collector = metrics.NewCollector()
)

func init() {

	prometheus.MustRegister(collector)
}

func main() {

	// Release Postgres Connection Pool
	defer metadb.ConnPool.Close()
	lis, err := net.Listen("tcp", ":50051")

	if err != nil {
		logging.PeppaMonLog(
			"fatal",
			"failed to bind Peppamon Telemetry TCP socket %v", err)

	}

	// Set gRPC Server Keepalive Settings
	grpcServerKeepalives := keepalive.ServerParameters{
		MaxConnectionIdle: 5 * time.Minute,
		Time:              30 * time.Second,
		Timeout:           time.Minute,
	}
	grpcServerKeepaliveOptions := grpc.KeepaliveParams(grpcServerKeepalives)

	// Create gRPC Server with options and middleware
	s := grpc.NewServer(
		grpcServerKeepaliveOptions,
		grpc.StreamInterceptor(grpcRecovery.StreamServerInterceptor()),
	)

	mdt_dialout.RegisterGRPCMdtDialoutServer(s, &HighObsSrv{exp: collector})

	logging.PeppaMonLog(
		"info",
		"Starting Peppamon Telemetry gRPC collector...")

	// Channel to handle graceful shutdown of GRPC Server
	ch := make(chan os.Signal, 1)

	// Write in Channel in case of OS request to shut process
	signal.Notify(ch, os.Interrupt)

	promHTTPSrv := http.Server{Addr: ":2112"}

	// Start Peppamon gRPC Telemetry Collector
	go func() {
		if err := s.Serve(lis); err != nil {

			logging.PeppaMonLog(
				"fatal",
				"Failed to start Peppamon Telemetry gRPC collector %v", err)

		}
	}()

	// Start Prometheus HTTP handler
	go func() {
		logging.PeppaMonLog(
			"info",
			"Starting Prometheus HTTP metrics handler...")

		http.Handle("/metrics", promhttp.Handler())

		http.HandleFunc("/telemetry-device", addTelemetryDeviceHandler)

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			_, errWelcomePage := w.Write([]byte(`<html>
             <head><title>Peppamon Cisco Telemetry Exporter</title></head>
             <body>
             <h1>Peppamon Cisco Telemetry Exporter</h1>
             <p><a href="/metrics">Metrics</a></p>
             </body>
             </html>`))

			if errWelcomePage != nil {
				logging.PeppaMonLog(
					"error",
					"Failed to render Welcome page %v", err)
			}
		})

		if err := promHTTPSrv.ListenAndServe(); err != http.ErrServerClosed {
			logging.PeppaMonLog(
				"fatal",
				"Failed to start Prometheus HTTP metrics handler %v", err)
		}
	}()

	// Block main function from exiting until ch receives value
	<-ch
	logging.PeppaMonLog("warning", "Shutting down Peppamon server...")

	// Stop Prom HTTP, GRPC server
	ctxPromHTTP, ctxCancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer ctxCancel()

	errPromHTTPShut := promHTTPSrv.Shutdown(ctxPromHTTP)

	if errPromHTTPShut != nil {
		logging.PeppaMonLog(
			"warning",
			"Error while shutting down Prometheus HTTP Server %v", errPromHTTPShut)
	}
	s.Stop()

}

func (s *HighObsSrv) MdtDialout(stream mdt_dialout.GRPCMdtDialout_MdtDialoutServer) error {

	var clientIPSocket string
	var telemetrySource metrics.Source

	// Extract gRPC client socket
	clientIPNet, ok := peer.FromContext(stream.Context())

	if ok {

		clientIPSocket = clientIPNet.Addr.String()
	}

	logging.PeppaMonLog(
		"info",
		"Client Socket %v sending gRPC Telemetry Stream...", clientIPSocket)

	// Make sure we only the Telemetry subscription once to avoid flooding stdout
	logFlag := false

	// Start Telemetry gRPC stream receive
	for {
		req, err := stream.Recv()

		if err == io.EOF {
			return status.Errorf(
				codes.OK, "OK")
		}

		// Handle client disconnection error
		if err != nil {
			logging.PeppaMonLog(
				"error",
				"Error while reading client %v stream: %v", clientIPSocket, err)

			// Removing Metrics from cache if client disconnected
			s.exp.Mutex.Lock()
			if _, ok := s.exp.Metrics[telemetrySource]; ok {

				delete(s.exp.Metrics, telemetrySource)
			}
			s.exp.Mutex.Unlock()

			return status.Errorf(
				codes.Aborted,
				"reading stream failed. Disconnecting now.",
			)
		}

		// Get gRPC stream data
		data := req.GetData()

		// msg is of type Cisco Telemetry
		msg := &telemetry.Telemetry{}

		// Unmarshal Telemetry protocol buffer message
		err = proto.Unmarshal(data, msg)

		if err != nil {
			logging.PeppaMonLog(
				"error",
				"Error while unmarshaling Proto message from client %v : %v", err, clientIPSocket)

			return status.Errorf(
				codes.Internal,
				"unable to unmarshal protocol buffer message",
			)
		}

		// The Metrics Source represents the metrics cache key and is a combination of the Telemetry NodeID
		// and YANG encoding path
		msgPath := msg.GetEncodingPath()
		telemetryNodeID := msg.GetNodeIdStr()
		telemetrySource = metrics.Source{NodeID: telemetryNodeID, Path: msgPath}

		// Instantiate Device Metrics Cache
		devMutex := &sync.Mutex{}
		deviceMetrics := &metrics.DeviceGroupedMetrics{Mutex: devMutex}

		// If Metric cache key already exists, invalidate and remove it
		// Otherwise dashboard may show arbitrary / constant values
		// See https://stackoverflow.com/questions/57304563/prometheus-exporter-direct-instrumentation-vs-custom-collector
		s.exp.Mutex.Lock()

		if _, ok := s.exp.Metrics[telemetrySource]; ok {
			delete(s.exp.Metrics, telemetrySource)
		}

		// Set the metric cache key to include the Telemetry NodeIF and the YANG encoding path
		s.exp.Metrics[telemetrySource] = deviceMetrics
		s.exp.Mutex.Unlock()

		// Limit logging of Telemetry client connections
		if !logFlag {
			logging.PeppaMonLog(
				"info",
				"Telemetry Subscription Request Received - Client %v - Node %v - YANG Model Path %v",
				clientIPSocket, msg.GetNodeIdStr(), msg.GetEncodingPath(),
			)
		}
		logFlag = true

		// Flag to determine whether the Telemetry device streams accepted YANG Node path
		yangPathSupported := false

		// Convert Proto Msg Timestamp to type Time for Prometheus metric
		timestamp := msg.GetMsgTimestamp()
		promTimestamp := time.Unix(int64(timestamp)/1000, 0)

		// Get Node ID from Telemetry Message
		node := msg.GetNodeIdStr()

		// Map Node IP Address to Hostname from KV Store
		ipAddr, err := net.ResolveIPAddr("ip", node)

		if err != nil {
			logging.PeppaMonLog("error", "unable to decode value %v as a valid IP Address with error %v", node, err)
			return status.Errorf(
				codes.InvalidArgument,
				fmt.Sprintf("Unable to decode value %v as a valid IP Address", node))
		}

		telemetryDevice, err := kvstore.LookupTelemetryHost(*ipAddr)

		if err != nil {
			logging.PeppaMonLog("error", "unable to lookup key %v in KV Config Store with error %v",
				ipAddr.IP.String(), err)

			return status.Errorf(
				codes.InvalidArgument,
				fmt.Sprintf("Unable to verify device %v in KV Store", node))
		}

		if _, ok := telemetryDevice["hostname"]; !ok {
			logging.PeppaMonLog("error", "unable to verify device %v in KV Config store", node)
			return status.Errorf(
				codes.InvalidArgument,
				fmt.Sprintf("Unable to verify device %v in KV Store", node))
		}

		for _, m := range metrics.CiscoMetricRegistrar {
			if msg.EncodingPath == m.EncodingPath {

				yangPathSupported = true

				go m.RecordMetricFunc(msg, deviceMetrics, promTimestamp.UTC(), telemetryDevice["hostname"])
			}
		}

		if !yangPathSupported {

			// TEMP : Log undesired YANG metrics path as JSON for development purpose
			j := jsonpb.Marshaler{}
			s, _ := j.MarshalToString(msg)
			ioutil.WriteFile("log.json", []byte(s), 0644)

			logging.PeppaMonLog(
				"error",
				"Received Telemetry message from client %v  (Device Name %v) for unsupported YANG Node Path %v",
				clientIPSocket, msg.GetNodeIdStr(), msg.GetEncodingPath())

			return status.Errorf(
				codes.InvalidArgument,
				fmt.Sprintf("YANG Node Path %v Telemetry subscription not supported", msg.GetEncodingPath()))
		}

	}

}

type telemetryDeviceKVStore struct {
	IPAddress string `json:"ipAddress"`
	Hostname  string `json:"hostname"`
}

func addTelemetryDeviceHandler(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	data := telemetryDeviceKVStore{}
	err := json.NewDecoder(r.Body).Decode(&data)

	if err != nil {
		logging.PeppaMonLog("error", "unable to decode JSON payload while creating new telemetry device: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ipAddr, err := net.ResolveIPAddr("ip", data.IPAddress)

	if err != nil {
		logging.PeppaMonLog("error", "unable to validate IP Address %v with error %v", data.IPAddress, err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if data.Hostname == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = kvstore.InsertNewTelemetryHost(*ipAddr, map[string]interface{}{
		"hostname": data.Hostname,
	})
	if err != nil {
		logging.PeppaMonLog("error", "unable to create host in KV Config Store with error %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}
