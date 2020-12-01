package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/niktheblak/temperature-api/internal/service"
)

type Server struct {
	Service *service.Service
}

func New(svc *service.Service) *Server {
	return &Server{Service: svc}
}

func (s *Server) Current(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	m, err := s.Service.Current(r.Context())
	if err != nil {
		log.Printf("Error while reading response: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(m); err != nil {
		log.Fatal(err)
	}
}
