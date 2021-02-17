package mutate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CoordAddr contains the address of the marblerun coordinator
var CoordAddr string

type patchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

// HandleMutate handles mutate requests and injects sgx tolerations into the request
func HandleMutate(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling mutate request")
	body := checkRequest(w, r)
	if body == nil {
		// Error was already written to w
		return
	}

	// mutate the request and add sgx tolerations to pod
	mutatedBody, err := mutateSimple(body, true)
	if err != nil {
		http.Error(w, "unable to mutate request", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(mutatedBody)
}

// HandleMutateNoSGX omits injecting sgx tolerations but otherwise functions the same as HandleMutate
func HandleMutateNoSGX(w http.ResponseWriter, r *http.Request) {
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

	// check if valid pod was sent
	var pod corev1.Pod
	if err := json.Unmarshal(admReviewReq.Request.Object.Raw, &pod); err != nil {
		return nil, err
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
			AuditAnnotations: map[string]string{
				"mutated": "true",
			},
		},
	}

	// create patch
	var patch []patchOperation

	// generate env variable values
	marbleType := pod.GetName()
	// ensure a name is set
	if marbleType == "" {
		marbleType = pod.GetGenerateName()
	}
	marbleDNSName := fmt.Sprintf("%s,%s.%s,%s.%s.svc.cluster.local", marbleType, marbleType, pod.GetNamespace(), marbleType, pod.GetNamespace())
	uuidFile := fmt.Sprintf("/%s/data/uuid", marbleType)

	// check if EDG env variables are set, if not set them here
	patch = append(patch, addEnvVar("EDG_MARBLE_COORDINATOR_ADDR", CoordAddr, pod.Spec.Containers)...)
	patch = append(patch, addEnvVar("EDG_MARBLE_TYPE", marbleType, pod.Spec.Containers)...)
	patch = append(patch, addEnvVar("EDG_MARBLE_DNS_NAMES", marbleDNSName, pod.Spec.Containers)...)
	patch = append(patch, addEnvVar("EDG_MARBLE_UUID_FILE", uuidFile, pod.Spec.Containers)...)

	// add sgx tolerations if enabled
	if sgx {
		patch = append(patch, patchOperation{
			Op:    "replace",
			Path:  "/spec/tolerations",
			Value: corev1.Toleration{Key: "kubernetes.azure.com/sgx_epc_mem_in_MiB"},
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

func envIsSet(envVar corev1.EnvVar, containers []corev1.Container) bool {
	for i := range containers {
		for _, setVar := range containers[i].Env {
			if setVar.Name == envVar.Name {
				return true
			}
		}
	}
	return false
}

// addEnvVar creates a patchOperation for a given env variable
func addEnvVar(envName string, envVal string, containers []corev1.Container) []patchOperation {
	var envPatch []patchOperation
	envVar := corev1.EnvVar{
		Name:  envName,
		Value: envVal,
	}
	if !envIsSet(envVar, containers) {
		for idx := range containers {
			envPatch = append(envPatch, patchOperation{
				Op:    "add",
				Path:  fmt.Sprintf("/spec/containers/%d/env/-", idx),
				Value: envVar,
			})
		}
	}
	return envPatch
}
