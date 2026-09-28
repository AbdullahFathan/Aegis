package response

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *ErrorBody `json:"error,omitempty"`
}

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JSON(w http.ResponseWriter, status int, payload Envelope) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}

func Success(w http.ResponseWriter, status int, data any) error {
	return JSON(w, status, Envelope{Success: true, Data: data})
}

func Error(w http.ResponseWriter, status int, code, message string) error {
	return JSON(w, status, Envelope{
		Success: false,
		Error:   &ErrorBody{Code: code, Message: message},
	})
}

func EncodeSuccess(data any) ([]byte, error) {
	return json.Marshal(Envelope{Success: true, Data: data})
}

func EncodeError(code, message string) ([]byte, error) {
	return json.Marshal(Envelope{
		Success: false,
		Error:   &ErrorBody{Code: code, Message: message},
	})
}
