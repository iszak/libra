package api

import (
	"github.com/ant0ine/go-json-rest/rest"
	"github.com/hashicorp/nomad/api"
	"github.com/prometheus/common/log"
	"github.com/underarmour/libra/nomad"
	"net/http"
)

type RestartRequest struct {
	Job   string `json:"job"`
	Group string `json:"group"`
	Task  string `json:"task"`
	Image string `json:"image"`
}

type RestartResponse struct {
	Eval string `json:"eval"`
}

func NewRestartRequest(job, group, task, image string) *RestartRequest {
	return &RestartRequest{
		Job:   job,
		Group: group,
		Task:  task,
		Image: image,
	}
}

func RestartHandler (n *api.Client) func(w rest.ResponseWriter, r *rest.Request) {
	return func(w rest.ResponseWriter, r *rest.Request) {
		var t RestartRequest
		err := r.DecodeJsonPayload(&t)
		if err != nil {
			log.Infoln("GOT AN ERROR")
			log.Errorln(err)
			rest.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		evalID, err := nomad.Restart(n, t.Job, t.Group, t.Task, t.Image)
		if err != nil {
			log.Error("Problem restarting the job " + err.Error())
			rest.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			log.Infoln("Restarted it! Evaluation " + evalID)
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "application/json; charset=UTF-8")
			respBody := &RestartResponse{
				Eval: evalID,
			}

			w.WriteJson(respBody)
		}
	}
}
