package mutate

import (
	"encoding/json"
	"errors"
	"fmt"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func mutateSimple(body []byte, sgx bool) ([]byte, error) {
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

	pT := v1.PatchTypeJSONPatch
	admReviewResponse.Response.PatchType = &pT

	var patch []map[string]interface{}

	patch = append(patch, map[string]interface{}{
		"op":    "replace",
		"path":  "/metadata/annotations/injected",
		"value": "success",
	})

	for idx := range pod.Spec.Containers {
		patch = append(patch, map[string]interface{}{
			"op":   "add",
			"path": fmt.Sprintf("/spec/containers/%d/env/-", idx),
			"value": map[string]string{
				"name":  "INJECTION",
				"value": "success",
			},
		})
	}

	var err error
	admReviewResponse.Response.Patch, err = json.Marshal(patch)
	if err != nil {
		return nil, err
	}

	admReviewResponse.Response.Result = &metav1.Status{
		Status: "Success",
	}

	bytes, err := json.Marshal(admReviewResponse)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}
