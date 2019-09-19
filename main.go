package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
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
}

var collector *metrics.Collector

func init() {
	// Register Peppamon Prometheus Collector
	collector = metrics.NewCollector()
	prometheus.MustRegister(collector)
}

func main() {

	// Release Postgres Connection Pool
	defer metadb.ConnPool.Close()
	lis, err := net.Listen("tcp", ":50051")

	if err != nil {
		logging.PeppaMonLog(
			"fatal",
			fmt.Sprintf("failed to bind Peppamon Telemetry TCP socket %v", err))

	}

	// Set gRPC Server Keepalive Settings
	grpcServerKeepalives := keepalive.ServerParameters{
		MaxConnectionIdle: 5 * time.Minute,
		Time:              10 * time.Second,
		Timeout:           30 * time.Second,
	}
	grpcServerOptions := grpc.KeepaliveParams(grpcServerKeepalives)

	s := grpc.NewServer(grpcServerOptions)

	mdt_dialout.RegisterGRPCMdtDialoutServer(s, &HighObsSrv{})

	logging.PeppaMonLog(
		"info",
		fmt.Sprint("Starting Peppamon Telemetry gRPC collector..."))

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
				fmt.Sprintf("Failed to start Peppamon Telemetry gRPC collector %v", err))

		}
	}()

	// Start Prometheus HTTP handler
	go func() {
		logging.PeppaMonLog(
			"info",
			fmt.Sprintf("Starting Prometheus HTTP metrics handler..."))

		http.Handle("/metrics", promhttp.Handler())

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
					fmt.Sprintf("Failed to render Welcome page %v", err))
			}
		})

		if err := promHTTPSrv.ListenAndServe(); err != http.ErrServerClosed {
			logging.PeppaMonLog(
				"fatal",
				fmt.Sprintf("Failed to start Prometheus HTTP metrics handler %v", err))
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
			fmt.Sprintf("Error while shutting down Prometheus HTTP Server %v", errPromHTTPShut))
	}
	s.Stop()

}

func (s *HighObsSrv) MdtDialout(stream mdt_dialout.GRPCMdtDialout_MdtDialoutServer) error {

	// Assign Collector Instance
	s.exp = collector

	var clientIPSocket string
	var telemetrySource metrics.Source

	// Extract gRPC client IP for logging purpose
	clientIPNet, ok := peer.FromContext(stream.Context())

	if ok {
		clientIPSocket = clientIPNet.Addr.String()
	}

	logging.PeppaMonLog(
		"info",
		fmt.Sprintf("Client Socket %v initiating gRPC Telemetry Stream...", clientIPSocket))

	// Make sure we only the Telemetry subscription once to avoid flooding stdout
	logFlag := false

	for {
		req, err := stream.Recv()

		if err == io.EOF {
			return nil
		}

		// Handle client disconnection error
		if err != nil {
			logging.PeppaMonLog(
				"error",
				fmt.Sprintf("Error while reading client %v stream: %v", clientIPSocket, err))

			// Removing Metrics from cache
			s.exp.Mutex.Lock()
			if _, ok := s.exp.Metrics[telemetrySource]; ok {

				delete(s.exp.Metrics, telemetrySource)
			}
			s.exp.Mutex.Unlock()

			return err
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
				fmt.Sprintf("Error while unmarshaling Proto message from client %v : %v", err, clientIPSocket))

			return err
		}

		// The Metrics Source represents the metrics cache key and is a combination of gRPC client socket
		// and YANG encoding path
		msgPath := msg.GetEncodingPath()
		clientIP := strings.Split(clientIPSocket, ":")
		telemetrySource = metrics.Source{Addr: clientIP[0], Path: msgPath}

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

		s.exp.Metrics[telemetrySource] = deviceMetrics
		s.exp.Mutex.Unlock()

		// Limit logging of Telemetry client connections
		if !logFlag {
			logging.PeppaMonLog(
				"info",
				fmt.Sprintf(
					"Telemetry Subscription Request Received - Client %v - Node %v - YANG Model Path %v",
					clientIPSocket, msg.GetNodeIdStr(), msg.GetEncodingPath(),
				),
			)
		}
		logFlag = true

		// Flag to determine whether the Telemetry device streams accepted YANG Node path
		yangPathSupported := false

		for _, m := range metrics.CiscoMetricRegistrar {
			if msg.EncodingPath == m.EncodingPath {

				yangPathSupported = true
				go m.RecordMetricFunc(msg, deviceMetrics)
			}
		}

		if !yangPathSupported {

			// TEMP : Log undesired YANG metrics path as JSON for development purpose
			j := jsonpb.Marshaler{}
			s, _ := j.MarshalToString(msg)
			ioutil.WriteFile("log.json", []byte(s), 0644)

			logging.PeppaMonLog(
				"error",
				fmt.Sprintf(
					"Received Telemetry message from client %v  (Device Name %v) for unsupported YANG Node Path %v",
					clientIPSocket, msg.GetNodeIdStr(), msg.GetEncodingPath()))

			return status.Errorf(
				codes.InvalidArgument,
				fmt.Sprintf("YANG Node Path %v Telemetry subscription not supported", msg.GetEncodingPath()))
		}

	}
}
