package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/edgelesssys/coordinator/coordinator/quote"
	"github.com/edgelesssys/coordinator/marble/marble"
)

const (
	Success             int = 0
	InternalError       int = 2
	AuthenticationError int = 4
	UsageError          int = 8
)

func main() {}

func premainTarget(argc int, argv []string, env []string) int {
	isServer := argc > 0 && argv[0] == "serve"
	tlsCertPem, tlsCertRaw, err := parsePemFromEnv(env, marble.EdgMarbleCert)
	if err != nil {
		log.Fatalf("failed to get TLS Certificate: %v", err)
	}
	_, rootCARaw, err := parsePemFromEnv(env, marble.EdgRootCA)
	if err != nil {
		log.Fatalf("failed to get root CA: %v", err)
	}
	_, privkRaw, err := parsePemFromEnv(env, marble.EdgMarblePrivKey)
	if err != nil {
		log.Fatalf("failed to get private key: %v", err)
	}

	// Verify certificate chain
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(rootCARaw) {
		return AuthenticationError
	}
	opts := x509.VerifyOptions{
		Roots:         roots,
		CurrentTime:   time.Now(),
		DNSName:       "localhost",
		Intermediates: x509.NewCertPool(),
	}
	tlsCert, err := x509.ParseCertificate(tlsCertPem.Bytes)
	if err != nil {
		return AuthenticationError
	}
	_, err = tlsCert.Verify(opts)
	if err != nil {
		log.Fatalf("failed to verify certificate chain: %v", err)
		return UsageError
	}

	// Run actual server-client application
	if isServer {
		runServer(tlsCertRaw, privkRaw, rootCARaw)
		return Success
	}
	err = runClient(tlsCertRaw, privkRaw, rootCARaw)
	if err != nil {
		log.Fatalf("failed to make connection to server: %v", err)
		return UsageError
	}
	return Success
}

func parsePemFromEnv(env []string, certName string) (*pem.Block, []byte, error) {
	certRaw := os.Getenv(certName)
	if len(certRaw) == 0 {
		return nil, nil, fmt.Errorf("could not find certificate in env")
	}
	certPem, _ := pem.Decode([]byte(certRaw))
	if certPem == nil {
		return nil, nil, fmt.Errorf("could not decode certificate in PEM format")
	}

	return certPem, []byte(certRaw), nil
}

func runServer(certRaw []byte, keyRaw []byte, rootCARaw []byte) {
	// generate server with TLSConfig
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(rootCARaw)
	tlsCert, err := tls.X509KeyPair(certRaw, keyRaw)
	if err != nil {
		log.Fatalf("cannot create TLS cert: %v", err)
		return
	}
	srv := &http.Server{
		Addr: "localhost:8080",
		TLSConfig: &tls.Config{
			ClientCAs:    roots,
			Certificates: []tls.Certificate{tlsCert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
		},
	}

	// handle '/' route
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Welcome to this Marbelous world!")
	})

	// run sever
	log.Fatal(srv.ListenAndServeTLS("", ""))
}

func runClient(certRaw []byte, keyRaw []byte, rootCARaw []byte) error {
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(rootCARaw)
	tlsCert, err := tls.X509KeyPair(certRaw, keyRaw)
	client := http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{
		RootCAs:      roots,
		Certificates: []tls.Certificate{tlsCert},
	}}}
	resp, err := client.Get("https://localhost:8080/")
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http.Get returned: %v", resp.Status)
	}
	log.Printf("Successful connection to Server: %v", resp.Status)
	return nil
}

func marbleTest(coordinationAddr, marbleType, marbleDNSNames string) int {
	// set env vars
	if err := os.Setenv(marble.EdgCoordinatorAddr, coordinationAddr); err != nil {
		log.Fatalf("failed to set env variable: %v", err)
		return InternalError
	}
	if err := os.Setenv(marble.EdgMarbleType, marbleType); err != nil {
		log.Fatalf("failed to set env variable: %v", err)
		return InternalError
	}

	if err := os.Setenv(marble.EdgMarbleDNSNames, marbleDNSNames); err != nil {
		log.Fatalf("failed to set env variable: %v", err)
		return InternalError
	}

	// call PreMain
	commonName := "marble" // Coordinator will assign an ID to us
	orgName := "Edgeless Systems GmbH"
	issuer := quote.NewERTIssuer()
	a, err := marble.NewAuthenticator(orgName, commonName, issuer)
	if err != nil {
		return InternalError
	}
	_, _, err = marble.PreMain(a, premainTarget)
	if err != nil {
		fmt.Println(err)
		return AuthenticationError
	}
	log.Println("Successfully authenticated with Coordinator!")
	return Success
}
