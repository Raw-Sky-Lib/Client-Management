package utils

import (
	"encoding/json"
	"net/http"
)

func RespondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func RespondError(w http.ResponseWriter, status int, message string) {
	RespondJSON(w, status, map[string]string{"error": message})
}

// ErrorResponse is the JSON body returned for all API errors. Used as Swagger response type.
type ErrorResponse struct {
	Error string `json:"error" example:"something went wrong"`
}

// MessageResponse is the JSON body for simple text success responses.
type MessageResponse struct {
	Message string `json:"message" example:"Check your email for a confirmation link."`
}

// OKResponse is the JSON body for simple boolean success responses.
type OKResponse struct {
	OK bool `json:"ok" example:"true"`
}
