package main

import (
	"crypto/tls"
	"encoding/base64"
	"log/slog"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/worker"

	"github.com/sagikazarmark/thesis/worker/activities/awsactivities"
	"github.com/sagikazarmark/thesis/worker/activities/kubeactivities"
	"github.com/sagikazarmark/thesis/worker/workflows"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	logger.Info("starting worker", slog.String("version", version), slog.String("revision", version), slog.String("revisionDate", version))

	var connectionOptions client.ConnectionOptions

	if lookupEnv("TEMPORAL_TLS", "false") == "true" {
		connectionOptions.TLS = loadTLS(logger)
	}

	temporalClient, err := client.Dial(client.Options{
		HostPort:          lookupEnv("TEMPORAL_ADDRESS", client.DefaultHostPort),
		Namespace:         lookupEnv("TEMPORAL_NAMESPACE", "default"),
		Logger:            log.NewStructuredLogger(logger.With(slog.String("subsystem", "temporal"))),
		ConnectionOptions: connectionOptions,
	})
	if err != nil {
		logger.Error("unable to create Temporal Client", slog.Any("error", err))
	}
	defer temporalClient.Close()

	w := worker.New(temporalClient, "thesis", worker.Options{})

	workflows.RegisterWorkflows(w)
	awsactivities.RegisterActivities(w)
	kubeactivities.RegisterActivities(w)

	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Error("unable to start Worker", slog.Any("error", err))

		os.Exit(1)
	}
}

func lookupEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

func loadTLS(logger *slog.Logger) *tls.Config {
	certFile := lookupEnv("TEMPORAL_TLS_CERT_FILE", "")
	keyFile := lookupEnv("TEMPORAL_TLS_KEY_FILE", "")

	certContent := lookupEnv("TEMPORAL_TLS_CERT", "")
	keyContent := lookupEnv("TEMPORAL_TLS_KEY", "")

	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			logger.Error("unable to load TLS files", slog.Any("error", err), slog.String("cert", certFile), slog.String("key", keyFile))

			os.Exit(1)
		}

		return &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	} else if certContent != "" && keyContent != "" {
		decodedCertContent, err := base64.StdEncoding.DecodeString(certContent)
		if err != nil {
			logger.Error("unable to decode certificate", slog.Any("error", err))

			os.Exit(1)
		}

		decodedKeyContent, err := base64.StdEncoding.DecodeString(keyContent)
		if err != nil {
			logger.Error("unable to decode key", slog.Any("error", err))

			os.Exit(1)
		}

		cert, err := tls.X509KeyPair(decodedCertContent, decodedKeyContent)
		if err != nil {
			logger.Error("unable to load TLS key pair", slog.Any("error", err))

			os.Exit(1)
		}

		return &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
	}

	logger.Error("TLS is enabled but no certificate is provided")

	os.Exit(1)

	return nil
}
