package webhook

import "testing"

func TestMutatesValidRequest(t *testing.T) {
	rawJSON := `{
"kind": "AdmissionReview",
"apiVersion": "admission.k8s.io/v1",
"request": {
	"TODO": "ADD VALID JSON ENTRY
}
}
`
	_, err := mutate([]byte(rawJSON), true)
	if err != nil {
		t.Errorf("failed to mutate request with error %s", err)
	}

	//TODO: Add checks for correct mutation including sgx values

	_, err = mutate([]byte(rawJSON), false)
	if err != nil {
		t.Errorf("failed to mutate request with error %s", err)
	}
	//TODO: Add checks for correct mutation without sgx values
}

func TestErrorsOnInvlad(t *testing.T) {
	rawJSON := `This should return Error`

	_, err := mutate([]byte(rawJSON), true)
	if err == nil {
		t.Error("did not fail on invalid request")
	}
}
