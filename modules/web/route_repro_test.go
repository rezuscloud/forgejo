package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func h(name, key string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(name + " " + key + "=" + chi.URLParam(req, key)))
	}
}

func TestRouteParamSiblingRepro(t *testing.T) {
	r := NewRoute()
	r.Group("/repos/{username}/{reponame}/actions", func() {
		r.Get("/tasks", h("TASKS", "reponame"))
		r.Group("/artifacts", func() {
			r.Get("", h("ARTS", "reponame"))
			r.Get("/{artifact_id}", h("ART", "artifact_id"))
		})
		r.Group("/jobs/{job_id}", func() {
			r.Get("", h("JOB", "job_id"))
			r.Get("/logs", h("JOBLOGS", "job_id"))
		})
		r.Group("/runs", func() {
			r.Get("", h("RUNS", "reponame"))
			r.Get("/index/{index}", h("BYINDEX", "index"))
			r.Get("/{run_id}", h("BYID", "run_id"))
			r.Delete("/{run_id}", h("DELRUN", "run_id"))
			r.Post("/{run_id}/cancel", h("CANCEL", "run_id"))
			r.Get("/{run_id}/jobs", h("RUNJOBS", "run_id"))
			r.Get("/{run_id}/logs", h("RUNLOGS", "run_id"))
			r.Get("/{run_id}/artifacts", h("RUNARTS", "run_id"))
		})
	})
	for path, expect := range map[string]string{
		"/repos/u/repo/actions/runs/index/5": "BYINDEX index=5",
		"/repos/u/repo/actions/runs/903":     "BYID run_id=903",
		"/repos/u/repo/actions/runs/903/jobs": "RUNJOBS run_id=903",
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		assert.Equal(t, expect, rec.Body.String(), path)
	}
}
