package mutate

import (
	"encoding/json"
	"strings"
	"testing"

	v1 "k8s.io/api/admission/v1"
)

func TestMutatesValidRequest(t *testing.T) {
	rawJSON := `{
		"apiVersion": "admission.k8s.io/v1",
		"kind": "AdmissionReview",
		"request": {
			"uid": "705ab4f5-6393-11e8-b7cc-42010a800002",
			"kind": {
				"group": "",
				"version": "v1",
				"kind": "Pod"
			},
			"resource": {
				"group": "",
				"version": "v1",
				"resource": "pods"
			},
			"requestKind": {
				"group": "",
				"version": "v1",
				"kind": "Pod"
			},
			"requestResource": {
				"group": "",
				"version": "v1",
				"resource": "pods"
			},
			"namespace": "injectable",
			"operation": "CREATE",
			"userInfo": {
				"username": "kubernetes-admin",
				"groups": [
					"system:masters",
					"system:authenticated"
				]
			},
			"object": {
				"kind": "Pod",
				"apiVersion": "v1",
				"metadata": {
					"name": "testpod",
					"namespace": "injectable",
					"creationTimestamp": null,
					"labels": {
						"name": "testpod"
						"marblerun.marbletype": "test"
					}
				},
				"spec": {
					"containers": [
						{
							"name": "testpod",
							"image": "test:image",
							"command": [
								"/bin/bash"
							],
							"terminationMessagePath": "/dev/termination-log",
							"terminationMessagePolicy": "File",
							"imagePullPolicy": "IfNotPresent"
						}
					],
					"restartPolicy": "Always",
					"terminationGracePeriodSeconds": 30,
					"dnsPolicy": "ClusterFirst",
					"serviceAccountName": "default",
					"serviceAccount": "default",
					"securityContext": {},
					"schedulerName": "default-scheduler",
					"priority": 0,
					"enableServiceLinks": true
				},
				"status": {}
			},
			"oldObject": null,
			"dryRun": false,
			"options": {
				"kind": "CreateOptions",
				"apiVersion": "meta.k8s.io/v1"
			}
		}
	}`

	CoordAddr = "coordinator-mesh-api.marblerun:25554"

	// test if patch contains all desired values
	response, err := mutate([]byte(rawJSON), true)
	if err != nil {
		t.Errorf("failed to mutate request with error %s", err)
	}
	r := v1.AdmissionReview{}
	if err := json.Unmarshal(response, &r); err != nil {
		t.Errorf("failed to unmarshal response with error %s", err)
	}
	if !strings.Contains(string(r.Response.Patch), `"op":"add","path":"/spec/containers/0/env","value":[{"name":"EDG_MARBLE_COORDINATOR_ADDR","value":"coordinator-mesh-api.marblerun:25554"}]`) {
		t.Error("failed to apply coordinator env variable patch")
	}
	if !strings.Contains(string(r.Response.Patch), `"op":"add","path":"/spec/containers/0/env/-","value":{"name":"EDG_MARBLE_TYPE","value":"testpod"}`) {
		t.Error("failed to apply marble type env variable patch")
	}
	if !strings.Contains(string(r.Response.Patch), `"op":"add","path":"/spec/containers/0/env/-","value":{"name":"EDG_MARBLE_DNS_NAMES","value":"testpod,testpod.injectable,testpod.injectable.svc.cluster.local"}`) {
		t.Error("failed to apply DNS name env varibale patch")
	}
	if !strings.Contains(string(r.Response.Patch), `"op":"add","path":"/spec/containers/0/env/-","value":{"name":"EDG_MARBLE_UUID_FILE","value":"/testpod/data/uuid"}`) {
		t.Error("failed to apply marble UUID file env variable patch")
	}
	if !strings.Contains(string(r.Response.Patch), `{"op":"add","path":"/spec/tolerations","value":{"key":"kubernetes.azure.com/sgx_epc_mem_in_MiB"}}`) {
		t.Error("failed to apply tolerations patch")
	}

	// test if patch works without sgx values
	response, err = mutate([]byte(rawJSON), false)
	if err != nil {
		t.Errorf("failed to mutate request with error %s", err)
	}
	if err := json.Unmarshal(response, &r); err != nil {
		t.Errorf("failed to unmarshal response with error %s", err)
	}
	if strings.Contains(string(r.Response.Patch), `{"op":"add","path":"/spec/tolerations","value":{"key":"kubernetes.azure.com/sgx_epc_mem_in_MiB"}}`) {
		t.Error("patch contained sgx tolerations, but tolerations were not supposed to be set")
	}
}

func TestPreSetValues(t *testing.T) {
	rawJSON := `{
		"apiVersion": "admission.k8s.io/v1",
		"kind": "AdmissionReview",
		"request": {
			"uid": "705ab4f5-6393-11e8-b7cc-42010a800002",
			"kind": {
				"group": "",
				"version": "v1",
				"kind": "Pod"
			},
			"resource": {
				"group": "",
				"version": "v1",
				"resource": "pods"
			},
			"requestKind": {
				"group": "",
				"version": "v1",
				"kind": "Pod"
			},
			"requestResource": {
				"group": "",
				"version": "v1",
				"resource": "pods"
			},
			"namespace": "injectable",
			"operation": "CREATE",
			"userInfo": {
				"username": "kubernetes-admin",
				"groups": [
					"system:masters",
					"system:authenticated"
				]
			},
			"object": {
				"kind": "Pod",
				"apiVersion": "v1",
				"metadata": {
					"name": "testpod",
					"namespace": "injectable",
					"creationTimestamp": null,
					"labels": {
						"name": "testpod"
					}
				},
				"spec": {
					"containers": [
						{
							"name": "testpod",
							"image": "test:image",
							"command": [
								"/bin/bash"
							],
							"terminationMessagePath": "/dev/termination-log",
							"terminationMessagePolicy": "File",
							"imagePullPolicy": "IfNotPresent",
							"env": [
								{
									"name": "EDG_MARBLE_COORDINATOR_ADDR",
									"value": "coordinator-mesh-api.marblerun:42"
								},
								{
									"name": "EDG_MARBLE_TYPE",
									"value": "different"
								},
								{
									"name": "EDG_MARBLE_DNS_NAMES",
									"value": "different.example.com"
								},
								{
									"name": "EDG_MARBLE_UUID_FILE",
									"value": "/different/data/unique/uuid"
								}
							]
						}
					],
					"restartPolicy": "Always",
					"terminationGracePeriodSeconds": 30,
					"dnsPolicy": "ClusterFirst",
					"serviceAccountName": "default",
					"serviceAccount": "default",
					"securityContext": {},
					"schedulerName": "default-scheduler",
					"priority": 0,
					"enableServiceLinks": true
				},
				"status": {}
			},
			"oldObject": null,
			"dryRun": false,
			"options": {
				"kind": "CreateOptions",
				"apiVersion": "meta.k8s.io/v1"
			}
		}
	}`

	CoordAddr = "coordinator-mesh-api.marblerun:25554"

	response, err := mutate([]byte(rawJSON), true)
	if err != nil {
		t.Errorf("failed to mutate request with error %s", err)
	}
	r := v1.AdmissionReview{}
	if err := json.Unmarshal(response, &r); err != nil {
		t.Errorf("failed to unmarshal response with error %s", err)
	}
	if strings.Contains(string(r.Response.Patch), `"op":"add","path":"/spec/containers/0/env","value":[{"name":"EDG_MARBLE_COORDINATOR_ADDR","value":"coordinator-mesh-api.marblerun:25554"}]`) {
		t.Error("applied coordinator env variable patch when it shouldnt have")
	}
	if strings.Contains(string(r.Response.Patch), `"op":"add","path":"/spec/containers/0/env/-","value":{"name":"EDG_MARBLE_TYPE","value":"testpod"}`) {
		t.Error("applied marble type env variable patch when it shouldnt have")
	}
	if strings.Contains(string(r.Response.Patch), `"op":"add","path":"/spec/containers/0/env/-","value":{"name":"EDG_MARBLE_DNS_NAMES","value":"testpod,testpod.injectable,testpod.injectable.svc.cluster.local"}`) {
		t.Error("applied DNS name env varibale patch when it shouldnt have")
	}
	if strings.Contains(string(r.Response.Patch), `"op":"add","path":"/spec/containers/0/env/-","value":{"name":"EDG_MARBLE_UUID_FILE","value":"/testpod/data/uuid"}`) {
		t.Error("applied marble UUID file env variable patch when it shouldnt have")
	}
}

func TestErrorsOnInvalid(t *testing.T) {
	rawJSON := `This should return Error`

	_, err := mutate([]byte(rawJSON), true)
	if err == nil {
		t.Error("did not fail on invalid request")
	}
}

func TestErrorsOnInvalidPod(t *testing.T) {
	rawJSON := `{
		"apiVersion": "admission.k8s.io/v1",
		"kind": "AdmissionReview",
		"request": {
			"object": "invalid"
		}
	}`
	_, err := mutate([]byte(rawJSON), true)
	if err == nil {
		t.Errorf("did not fail when sending invalid request with error %s", err)
	}
}
