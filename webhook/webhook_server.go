package webhook

import (
	"net/http"

	"github.com/edgelesssys/marblerun/util"
)

var coordAddr string

func main() {
	certPath := util.MustGetenv("WEBHOOK_CRT")
	keyPath := util.MustGetenv("WEBHOOK_KEY")
	coordAddr = util.MustGetenv("EDG_MARBLE_COORDINATOR_ADDR")
	mux := http.NewServeMux()

	mux.HandleFunc("/mutate", handleMutate)
	mux.HandleFunc("/mutate-no-sgx", handleMutateNoSGX)

	s := &http.Server{
		// Addresse forwarding to 443 should be handled by the webhook service object
		Addr:    ":8443",
		Handler: mux,
	}

	// TODO: Add logging maybe
	s.ListenAndServeTLS(certPath, keyPath)
}
