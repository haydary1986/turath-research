package model

import (
	"encoding/json"
	"net/http"
)

// APIResponse is the standard envelope for all API responses.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
	Total  int `json:"total,omitempty"`
	Page   int `json:"page,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	resp := APIResponse{
		Success: status >= 200 && status < 300,
		Data:    data,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func RespondError(w http.ResponseWriter, status int, message string) {
	resp := APIResponse{
		Success: false,
		Error:   message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func RespondPaginated(w http.ResponseWriter, status int, data interface{}, total, page, limit int) {
	resp := APIResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Total: total,
			Page:  page,
			Limit: limit,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}
