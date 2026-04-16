package httprouter

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"Flatly/internal/auth"
	"Flatly/internal/flat"
	"Flatly/internal/house"
)

func NewRouter(authHandler *auth.HTTPHandler, houseHandler *house.HTTPHandler, flatHandler *flat.HTTPHandler, authService *auth.Service) http.Handler {
	r := chi.NewRouter()

	r.Post("/dummyLogin", authHandler.HandleDummyLogin)
	r.Post("/register", authHandler.HandleRegister)
	r.Post("/login", authHandler.HandleLogin)

	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(authService))

		r.With(auth.RequireRole(auth.RoleModerator)).Post("/house/create", houseHandler.HandleCreate)
		r.Post("/flat/create", flatHandler.HandleCreate)
		r.With(auth.RequireRole(auth.RoleModerator)).Post("/flat/update", flatHandler.HandleUpdate)
		r.Get("/house/{id}", houseHandler.HandleGetByID)
		r.With(auth.RequireRole(auth.RoleClient)).Post("/house/{id}/subscribe", houseHandler.HandleSubscribe)
	})

	return r
}
