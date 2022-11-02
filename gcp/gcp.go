package gcp

import (
	"encoding/json"
	"log"
)

// https://cloud.google.com/functions/docs/monitoring/logging

type (
	LogEntry struct {
		Message  string `json:"message"`
		Severity string `json:"severity,omitempty"`
		Trace    string `json:"logging.googleapis.com/trace,omitempty"`

		// Logs Explorer allows filtering and display of this as `jsonPayload.component`.
		Component string `json:"component,omitempty"`
	}
)

// String renders an entry structure to the JSON format expected by Cloud Logging.
func (e LogEntry) String() string {
	if e.Severity == "" {
		e.Severity = "INFO"
	}
	out, err := json.Marshal(e)
	if err != nil {
		log.Printf("json.Marshal: %v", err)
	}
	return string(out)
}
