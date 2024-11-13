package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// In -> request body
// Out -> response body
type TargetFunc[In any, Out any] func(*http.Request, In) (*Out, error)

func HandleBody[In any, Out any](f TargetFunc[In, Out]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var in In

		// Retrieve data from request.
		err := json.NewDecoder(req.Body).Decode(&in)
		if err != nil {
			// Format error response
			fmt.Printf("err1: %v\n", err)
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		// Call out to target function
		out, err := f(req, in)
		if err != nil {
			// Format error response
			fmt.Printf("err2: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		SendResponse(w, out)
	})
}

func SendResponse(w http.ResponseWriter, response interface{}) {
	if response != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		if json.NewEncoder(w).Encode(response) != nil {
			log.Printf("failed to encode response: %v", response)
			return
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
