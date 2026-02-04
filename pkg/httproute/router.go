package httprouter

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"Flatly/internal/user"
)

func NewRouter(userHandler *user.HTTPHandler, logger *zap.Logger) http.Handler {
	r := chi.NewRouter()

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/users", userHandler.HandleCreateUser)
	})

	return r
}
