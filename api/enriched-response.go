package api

import (
	"net/http"
)

// EnrichedResponseWriter is a wrapper for http.ResponseWriter for better information extraction.
type EnrichedResponseWriter struct {
	http.ResponseWriter
	Status int
}

// NewEnrichedResponseWriter wraps a http.ResponseWriter.
func NewEnrichedResponseWriter(w http.ResponseWriter) *EnrichedResponseWriter {
	return &EnrichedResponseWriter{
		w,
		0,
	}
}

// WriteHeader wraps the original WriteHeader method to extract information.
func (ew *EnrichedResponseWriter) WriteHeader(code int) {
	ew.Status = code
	ew.ResponseWriter.WriteHeader(code)
}
