package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type response struct {
	Data any  `json:"data"`
	Meta *meta `json:"meta,omitempty"`
}

type meta struct {
	Total   int `json:"total"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeData(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, response{Data: data})
}

func writeDataPaginated(w http.ResponseWriter, data any, total, page, perPage int) {
	writeJSON(w, http.StatusOK, response{
		Data: data,
		Meta: &meta{Total: total, Page: page, PerPage: perPage},
	})
}

func writeError(w http.ResponseWriter, status int, errType, message string) {
	writeJSON(w, status, errorResponse{
		Error: errorBody{Type: errType, Message: message},
	})
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}
