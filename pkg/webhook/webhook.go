// Package webhook holds a handler dealing with the mutating webhook admission controller
package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/jaegertracing/jaeger-operator/pkg/inject"
)

// Handler represents the webhook HTTP handler
type Handler struct {
	client client.Client
}

// NewHandler creates a new webhook HTTP handler
func NewHandler(client client.Client) *Handler {
	return &Handler{client: client}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Debug("received webhook call")

	codecs := serializer.NewCodecFactory(runtime.NewScheme())
	deserializer := codecs.UniversalDeserializer()

	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		log.Warn("received a webhook call, but the request body was empty.")
		http.Error(w, "Admission review not provided", http.StatusBadRequest)
		return
	}

	log.WithField("body", string(body)).Debug("webhook content")

	var admissionResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		log.WithError(err).Warn("can't decode body")
		admissionResponse = &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
	} else {
		log.WithField("raw", string(ar.Request.Object.Raw)).Debug("processing admission review")
		if admissionResponse, err = inject.Process(&ar, h.client); err != nil {
			log.WithFields(log.Fields{
				"name":        ar.Request.Name,
				"namespace":   ar.Request.Namespace,
				"operation":   ar.Request.Operation,
				"resource":    ar.Request.Resource,
				"subResource": ar.Request.SubResource,
			}).WithError(err).Warn("can't process the admission review")
			admissionResponse = &v1beta1.AdmissionResponse{
				Result: &metav1.Status{
					Message: err.Error(),
				},
			}
		}
	}

	if admissionResponse != nil {
		ar.Response = admissionResponse
		if ar.Request != nil {
			ar.Response.UID = ar.Request.UID
		}
	}

	log.WithFields(log.Fields{
		"admissionResponse": ar.Response,
		"patch":             string(ar.Response.Patch),
	}).Debug("returning admission review response")

	resp, err := json.Marshal(ar)
	if err != nil {
		log.WithError(err).Error("can't encode response")
		http.Error(w, fmt.Sprintf("could not encode response: %v", err), http.StatusInternalServerError)
	}

	if _, err := w.Write(resp); err != nil {
		log.WithError(err).Error("can't write response")
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
