package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"

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
	w.Header().Set("ETag", etag(m))
	if err := json.NewEncoder(w).Encode(m); err != nil {
		log.Fatal(err)
	}
}

func etag(measurements map[string]service.Measurement) string {
	if len(measurements) == 0 {
		return ""
	}
	var timestamps []float64
	for _, m := range measurements {
		timestamps = append(timestamps, float64(m.Timestamp.Unix()))
	}
	sort.Float64s(timestamps)
	newest := int64(timestamps[len(timestamps)-1])
	return strconv.FormatInt(newest, 16)
}
