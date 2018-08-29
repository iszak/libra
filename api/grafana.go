package api

import (
	"encoding/json"
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/hashicorp/nomad/api"
	"github.com/prometheus/common/log"
	"github.com/underarmour/libra/nomad"
	"net/http"
)

type GrafanaRequest struct {
	Title       string               `json:"title"`
	State       string               `json:"state"`
	Count       int                  `json:"count"`
	Message     string               `json:"message"`
	EvalMatches []GrafanaEvalMatches `json:"evalMatches"`
}

type GrafanaEvalMatches struct {
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
}

type GrafanaMessageBody struct {
	Job            string  `json:"job"`
	Group          string  `json:"group"`
	MinCount       int     `json:"min_count"`
	MaxCount       int     `json:"max_count"`
	MaxThreshold   float64 `json:"max_threshold"`
	MinThreshold   float64 `json:"min_threshold"`
	MaxAction      string  `json:"max_action"`
	MinAction      string  `json:"min_action"`
	MaxActionCount int     `json:"max_action_count"`
	MinActionCount int     `json:"min_action_count"`
}

func GrafanaHandler (n *api.Client) func(w rest.ResponseWriter, r *rest.Request) {
	return func(w rest.ResponseWriter, r *rest.Request) {
		var t GrafanaRequest
		err := r.DecodeJsonPayload(&t)
		if err != nil {
			log.Errorln(err)
			rest.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var mb GrafanaMessageBody
		if err := json.Unmarshal([]byte(t.Message), &mb); err != nil {
			log.Errorf("Problem parsing Grafana webhook json %s: %s", t.Message, err)
			rest.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		log.Infof("Received Grafana webhook: %v", t.Message)

		var amount int
		// TODO: Right now this only grabs the first match. Really, we should take all of them and average them together
		if len(t.EvalMatches) == 0 {
			log.Infof("Alert %s has been cleared. Doing nothing...", t.Title)
			return
		}
		if t.EvalMatches[0].Value < mb.MinThreshold {
			amount = -mb.MinActionCount
		} else if t.EvalMatches[0].Value > mb.MaxThreshold {
			amount = mb.MaxActionCount
		} else {
			w.WriteHeader(http.StatusOK)
			return
		}

		evalID, newCount, err := nomad.Scale(n, mb.Job, mb.Group, amount, mb.MinCount, mb.MaxCount)
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
