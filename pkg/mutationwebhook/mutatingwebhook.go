package mutationwebhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/noaabarki/cel-cosign-poc/pkg/provider"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

func HandleMutate(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
		return
	}

	// Decode the admission request
	deserializer := serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer()
	ar := admissionv1.AdmissionReview{}
	_, _, err = deserializer.Decode(body, nil, &ar)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to decode admission request: %v", err), http.StatusBadRequest)
		return
	}

	if ar.Request == nil {
		http.Error(w, "Invalid request type", http.StatusBadRequest)
		return
	}

	fmt.Println("Received request to mutate pod:", ar.Request.Name)

	// Call the mutate function to modify the pod
	mutatedPod := mutate(ar.Request.Object.Raw, r.Context())

	// Create the admission response
	admissionResponse := admissionv1.AdmissionResponse{
		Allowed: true,
		Result: &metav1.Status{
			Status: metav1.StatusSuccess,
		},
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch
			return &pt
		}(),
		Patch: mutatedPod,
	}

	// Create the admission review response
	ar.Response = &admissionv1.AdmissionResponse{
		UID:     ar.Request.UID,
		Allowed: admissionResponse.Allowed,
		Result:  &metav1.Status{Status: metav1.StatusSuccess},
	}

	// Encode the response
	resp, err := json.Marshal(ar)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode admission response: %v", err), http.StatusInternalServerError)
		return
	}

	// Write the response
	if _, err := w.Write(resp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to write response: %v", err), http.StatusInternalServerError)
		return
	}
}

func mutate(raw []byte, ctx context.Context) []byte {
	// Decode the raw pod into a Pod object
	deployment := v1.Deployment{}
	if err := json.Unmarshal(raw, &deployment); err != nil {
		return nil
	}

	// Modify the pod as needed
	images := []string{}
	for _, container := range deployment.Spec.Template.Spec.Containers {
		images = append(images, container.Image)
	}

	isValid, err := provider.VerifyImages(images, ctx)
	if !isValid || err != nil {
		// add annotation to deployment
		deployment.ObjectMeta.Annotations["cosign.datree.io/valid"] = "false"
	}

	// Encode the mutated pod
	mutatedDeployment, _ := json.Marshal(deployment)
	return mutatedDeployment
}
