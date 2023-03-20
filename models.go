package main

import (
	"encoding/json"
	"net/http"
)

// Book represents a book entity.
type Book struct {
	ID          string `json:"id" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description" binding:"required"`
	Author      string `json:"author" binding:"required"`
	Price       string `json:"price" binding:"required"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// APIError is the data model sent when an error occurred during request processing.
type APIError struct {
	RequestID string      `json:"requestid"`
	Status    int         `json:"status"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
}

// APIError is the data model sent when a request succeed.
type APIResponse struct {
	RequestID string      `json:"requestid"`
	Status    int         `json:"status"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
}

func NewAPIError(requestid string, status int, message string, data interface{}) *APIError {
	return &APIError{
		RequestID: requestid,
		Status:    status,
		Message:   message,
		Data:      data,
	}
}

func GenericResponse(requestid string, status int, message string, data interface{}) *APIResponse {
	return &APIResponse{
		RequestID: requestid,
		Status:    status,
		Message:   message,
		Data:      data,
	}
}

func WriteErrorResponse(w http.ResponseWriter, errResp *APIError) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(errResp.Status)
	return json.NewEncoder(w).Encode(errResp)
}

func WriteResponse(w http.ResponseWriter, resp *APIResponse) error {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(resp.Status)
	return json.NewEncoder(w).Encode(resp)
}
