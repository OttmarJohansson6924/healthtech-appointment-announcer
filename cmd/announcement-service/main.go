package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"example.com/healthtech-announcer/internal/announcement"
)

func main() {
	apiKey := os.Getenv("INFRAI_API_KEY")
	if apiKey == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}

	service := announcement.NewService(announcement.NewInfraiClient(apiKey, nil))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /appointment-changes", func(w http.ResponseWriter, r *http.Request) {
		var change announcement.Change
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&change); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid appointment change"})
			return
		}

		result, err := service.Handle(r.Context(), change)
		if err != nil {
			status := http.StatusBadGateway
			var apiErr *announcement.InfraiError
			switch {
			case errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500:
				status = apiErr.HTTPStatus
			case !errors.As(err, &apiErr):
				status = http.StatusUnprocessableEntity
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, result)
	})

	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5_000_000_000}
	log.Printf("appointment announcement service listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
