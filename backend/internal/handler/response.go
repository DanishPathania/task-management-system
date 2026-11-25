package handler

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func RespondError(w http.ResponseWriter, statusCode int, errCode, message string) {
	JSON(w, statusCode, ErrorResponse{
		Error:   errCode,
		Message: message,
	})
}
