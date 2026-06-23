package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/fleetops/maintenance/internal/handler/dto"
)

// Recovery returns an HTTP middleware that catches panics and returns a 500
// Internal Server Error with a structured JSON response.
//
// [Archetype Convention Addition] — Error Handling (ISO/IEC 25010 Reliability)
func Recovery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)

					errResp := dto.ErrorResponse{
						Error:   "internal_server_error",
						Message: "An unexpected error occurred",
						Code:    http.StatusInternalServerError,
					}

					_ = json.NewEncoder(w).Encode(errResp)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
