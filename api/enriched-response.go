package api

import (
	"net/http"
)

type EnrichedResponseWriter struct {
	http.ResponseWriter
	Status int
}

func NewEnrichedResponseWriter(w http.ResponseWriter) *EnrichedResponseWriter {
	return &EnrichedResponseWriter{
		w,
		0,
	}
}

func (ew *EnrichedResponseWriter) WriteHeader(code int) {
	ew.Status = code
	ew.ResponseWriter.WriteHeader(code)
}
