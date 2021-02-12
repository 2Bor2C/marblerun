package webhook

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	body := checkRequest(w, r)
	if body == nil {
		// Error was already written to w
		return
	}

	// mutate the request and add sgx tolerations to pod
	mutatedBody, err := mutate(body, true)
	if err != nil {
		http.Error(w, "unable to mutate request", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(mutatedBody)
}

// same as handleMutate but apply mutate without sgx values
func handleMutateNoSGX(w http.ResponseWriter, r *http.Request) {
	body := checkRequest(w, r)
	if body == nil {
		// Error was already written to w
		return
	}

	// mutate the request and omit sgx tolerations
	mutatedBody, err := mutate(body, false)
	if err != nil {
		http.Error(w, "unable to mutate request", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(mutatedBody)
}

// mutate handles the creation of json patches for pods
func mutate(body []byte, sgx bool) ([]byte, error) {
	admReviewReq := v1.AdmissionReview{}
	if err := json.Unmarshal(body, &admReviewReq); err != nil {
		return nil, err
	}

	if admReviewReq.Request == nil {
		return nil, errors.New("empty admission request")
	}

	// admission response
	admReviewResponse := v1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &v1.AdmissionResponse{
			Allowed: true,
			UID:     admReviewReq.Request.UID,
		},
	}

	// create patch
	var patch []patchOperation
	if sgx {
		// TODO: inject variables here including sgx
		patch = append(patch, patchOperation{
			Op:    "replace",
			Path:  "/spec/tolerations",
			Value: corev1.Toleration{Key: "kubernetes.azure.com/sgx_epc_mem_in_MiB"},
		})
	} else {
		patch = append(patch, patchOperation{
			// TODO: inject variables here but omit sgx values
		})
	}

	// convert admission response into bytes and return
	var err error
	admReviewResponse.Response.Patch, err = json.Marshal(patch)
	if err != nil {
		return nil, err
	}
	bytes, err := json.Marshal(&admReviewResponse)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// check if http was POST and not empty
func checkRequest(w http.ResponseWriter, r *http.Request) []byte {
	if r.Method != http.MethodPost {
		http.Error(w, "unable to handle requests other than POST", http.StatusBadRequest)
		return nil
	}

	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		http.Error(w, "wrong application type", http.StatusBadRequest)
		return nil
	}

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, "unable to read request", http.StatusBadRequest)
		return nil
	}

	return body
}
