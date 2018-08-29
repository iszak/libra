package api

import (
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/hashicorp/nomad/api"
	"github.com/prometheus/common/log"
	"github.com/underarmour/libra/config"
	"github.com/underarmour/libra/nomad"
	"net/http"
)

type ScaleRequest struct {
	Job   string `json:"job"`
	Group string `json:"group"`
	Count int    `json:"count"`
}

type ScaleResponse struct {
	Eval     string `json:"eval"`
	NewCount int    `json:"new_count"`
}

func NewScaleRequest(job, group string, count int) *ScaleRequest {
	return &ScaleRequest{
		Job:   job,
		Group: group,
		Count: count,
	}
}

func ScaleHandler (c *config.RootConfig, n *api.Client) func(w rest.ResponseWriter, r *rest.Request) {
	return func(w rest.ResponseWriter, r *rest.Request) {
		var t ScaleRequest
		err := r.DecodeJsonPayload(&t)
		if err != nil {
			log.Errorln(err)
			rest.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if t.Count == 0 {
			log.Error("Amount to increment or decrement cannot be 0.")
			rest.Error(w, err.Error(), http.StatusInternalServerError)
		}
		configGroup := c.Jobs[t.Job].Groups[t.Group]
		evalID, newCount, err := nomad.Scale(n, t.Job, t.Group, t.Count, configGroup.MinCount, configGroup.MaxCount)
		if err != nil {
			log.Error("Problem scaling the task group " + err.Error())
			rest.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			log.Infoln("Scaled it! Evaluation " + evalID)
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			respBody := &ScaleResponse{
				Eval:     evalID,
				NewCount: newCount,
			}

			w.WriteJson(respBody)
		}
	}

}
