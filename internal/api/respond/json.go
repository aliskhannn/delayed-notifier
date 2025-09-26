package respond

import (
	"encoding/json"
	"net/http"

	"github.com/wb-go/wbf/zlog"
)

// Success represents a standard structure for successful responses.
type Success struct {
	Result interface{} `json:"result"`
}

// Error represents a standard structure for error responses.
type Error struct {
	Message string `json:"message"`
}

// JSON sends a JSON response with the given HTTP status code and data.
//
// It sets the "Content-Type" header to "application/json" and encodes
// the provided data into JSON format. Logs any encoding errors.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Encode data to JSON and write to response.
	if err := json.NewEncoder(w).Encode(data); err != nil {
		zlog.Logger.Error().Err(err).Interface("data", data).Msg("failed to encode JSON response")
	}
}

// OK sends a 200 OK response with the given result payload.
//
// The result is wrapped in a Success struct.
func OK(w http.ResponseWriter, result interface{}) {
	JSON(w, http.StatusOK, Success{Result: result})
}

// Created sends a 201 Created response with the given result payload.
//
// The result is wrapped in a Success struct.
func Created(w http.ResponseWriter, result interface{}) {
	JSON(w, http.StatusCreated, Success{Result: result})
}

// Fail sends an error response with the specified HTTP status code.
//
// The error message is wrapped in an Error struct.
func Fail(w http.ResponseWriter, status int, err error) {
	JSON(w, status, Error{Message: err.Error()})
}
