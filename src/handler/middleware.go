package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
)

type SDTargetsMiddleware struct {
	SDTargets *SDTargets
	SDGroup   string
	Client    *redis.Client
	Context   context.Context
	ApiToken  string
	SdToken   string
	TTL       int
}

type HttpSD struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

type StaticConfigDocument struct {
	SDTarget StaticConfig `json:"static_config"`
	SDGroup  string       `json:"target_group"`
}

type IDDocument struct {
	ID uuid.UUID `json:"id"`
}

func (s *SDTargetsMiddleware) HandleSDTarget(w http.ResponseWriter, r *http.Request) {
	if !s.isApiTokenValid(r) {
		s.forbiddenResponse(w)
		return
	} else {
		switch strings.ToUpper(r.Method) {
		case "GET":
			w.Header().Set("Content-Type", "application/json")
			s.handleGetByID(w, r)
		case "PUT":
			w.Header().Set("Content-Type", "application/json")
			s.handleInsert(w, r)
		case "DELETE":
			w.Header().Set("Content-Type", "application/json")
			s.handleDelete(w, r)
		}
	}
}

func (s *SDTargetsMiddleware) HandleDiscover(w http.ResponseWriter, r *http.Request) {
	if s.SdToken != "" {
		if !s.isSdTokenValid(r) {
			s.forbiddenResponse(w)
			return
		}
	}
	switch strings.ToUpper(r.Method) {
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		s.handleGetAll(w, r)
	}
}

func (s *SDTargetsMiddleware) HandleDiscoverGroup(w http.ResponseWriter, r *http.Request) {
	if s.SdToken != "" {
		if !s.isSdTokenValid(r) {
			s.forbiddenResponse(w)
			return
		}
	}
	switch strings.ToUpper(r.Method) {
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		s.handleGetByGroupName(w, r)
	}
}

func (s *SDTargetsMiddleware) handleGetAll(w http.ResponseWriter, r *http.Request) {
	res := []HttpSD{}
	targets, _ := s.SDTargets.Scan(s.Context, s.Client)
	for _, target := range targets.Items {
		res = append(res, HttpSD{target.Targets, target.Labels})
	}
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *SDTargetsMiddleware) handleGetByGroupName(w http.ResponseWriter, r *http.Request) {
	res := []HttpSD{}

	grp := r.URL.Query().Get("name")
	if grp == "" {
		http.Error(w, "Get parameter `name` is not defined", http.StatusBadRequest)
		return
	}
	targets, _ := s.SDTargets.Scan(s.Context, s.Client)
	for _, target := range targets.Items {
		if target.Group == grp {
			res = append(res, HttpSD{target.Targets, target.Labels})
		}
	}
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *SDTargetsMiddleware) handleInsert(w http.ResponseWriter, r *http.Request) {
	var req StaticConfigDocument
	var res IDDocument
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, e := s.SDTargets.Insert(req.SDTarget, s.Context, s.Client, s.TTL)
	if e != nil {
		http.Error(w, e.Error(), http.StatusBadRequest)
		return
	}
	res = IDDocument{ID: id}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (s *SDTargetsMiddleware) handleDelete(w http.ResponseWriter, r *http.Request) {
	var req IDDocument
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, e := s.SDTargets.Delete(req.ID, s.Context, s.Client)
	if e != nil {
		http.Error(w, e.Error(), http.StatusBadRequest)
		return
	}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *SDTargetsMiddleware) handleGetByID(w http.ResponseWriter, r *http.Request) {
	var req IDDocument
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rel, err := s.SDTargets.Retrieve(req.ID, s.Context, s.Client)
	if err == ErrIDNotFound {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res := StaticConfigDocument{SDTarget: rel}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *SDTargetsMiddleware) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}

func (s *SDTargetsMiddleware) isApiTokenValid(r *http.Request) bool {
	token := r.Header.Get("x-api-token")
	if token != s.ApiToken {
		return false
	} else {
		return true
	}
}

func (s *SDTargetsMiddleware) isSdTokenValid(r *http.Request) bool {
	token := r.Header.Get("x-sd-token")
	if token != s.SdToken {
		return false
	} else {
		return true
	}
}

func (s *SDTargetsMiddleware) forbiddenResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusForbidden)
	io.WriteString(w, "FORBIDDEN")
}

func FlushBufferOnShutdown(shutdownWaiter *sync.WaitGroup) {
	shutdownWaiter.Done()
}
