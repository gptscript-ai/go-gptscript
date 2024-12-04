package daemon

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
)

// CreateServer creates a new HTTP server with TLS configured for GPTScript.
// This function should be used when creating a new server for a daemon tool.
// The server should then be started with the StartServer function.
func CreateServer() (*http.Server, error) {
	tlsConfig, err := getTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get TLS config: %v\n", err)
	}

	server := &http.Server{
		Addr:      fmt.Sprintf("127.0.0.1:%s", os.Getenv("PORT")),
		TLSConfig: tlsConfig,
	}
	return server, nil
}

// StartServer starts an HTTP server created by the CreateServer function.
// This is for use with daemon tools.
func StartServer(server *http.Server) error {
	if err := server.ListenAndServeTLS("", ""); err != nil {
		return fmt.Errorf("stopped serving: %v\n", err)
	}
	return nil
}

func getTLSConfig() (*tls.Config, error) {
	certB64 := os.Getenv("CERT")
	privateKeyB64 := os.Getenv("PRIVATE_KEY")
	gptscriptCertB64 := os.Getenv("GPTSCRIPT_CERT")

	if certB64 == "" {
		return nil, fmt.Errorf("CERT not set")
	} else if privateKeyB64 == "" {
		return nil, fmt.Errorf("PRIVATE_KEY not set")
	} else if gptscriptCertB64 == "" {
		return nil, fmt.Errorf("GPTSCRIPT_CERT not set")
	}

	certBytes, err := base64.StdEncoding.DecodeString(certB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cert base64: %v\n", err)
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key base64: %v\n", err)
	}

	gptscriptCertBytes, err := base64.StdEncoding.DecodeString(gptscriptCertB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode gptscript cert base64: %v\n", err)
	}

	cert, err := tls.X509KeyPair(certBytes, privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create X509 key pair: %v\n", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(gptscriptCertBytes) {
		return nil, fmt.Errorf("failed to append gptscript cert to pool")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    pool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}, nil
}
