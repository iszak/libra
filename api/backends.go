package api

import (
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/underarmour/libra/backend"
)

type BackendResponse struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

func BackendsHandler(backends backend.ConfiguredBackends) func(w rest.ResponseWriter, r *rest.Request) {
	return func(w rest.ResponseWriter, r *rest.Request) {
		backendResponses := []BackendResponse{}
		for name, cbe := range backends {
			newBV := BackendResponse{
				Name: name,
				Kind: cbe.Kind,
			}
			backendResponses = append(backendResponses, newBV)
		}

		w.WriteJson(backendResponses)
	}
}
