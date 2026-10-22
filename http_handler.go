package legalmail

import (
	"encoding/json"
	"errors"
	"net/http"
)

func NewHandler(sender *ReceiptSender) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notices", func(w http.ResponseWriter, r *http.Request) {
		var notice Notice
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&notice); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON request"})
			return
		}
		delivery, err := sender.Dispatch(r.Context(), notice)
		if err != nil {
			status := http.StatusUnprocessableEntity
			var apiErr *InfraiError
			if errors.As(err, &apiErr) {
				status = http.StatusBadGateway
				if apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500 {
					status = apiErr.HTTPStatus
				}
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, delivery)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
