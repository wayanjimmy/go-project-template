package server

import (
	"go-project-template/logger"
	v1 "go-project-template/rest/v1"
	"go-project-template/service"

	"github.com/go-chi/chi/v5"
)

func SetupRouter(userService service.UserService, searchService service.SearchService, log *logger.Logger) *chi.Mux {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware)

	userHandler := v1.NewUserHandler(userService, log)
	searchHandler := v1.NewSearchHandler(searchService)

	r.Route("/v1", func(r chi.Router) {
		r.Post("/users", userHandler.Create)
		r.Get("/users/{id}", userHandler.Get)
		r.Put("/users/{id}", userHandler.Update)
		r.Delete("/users/{id}", userHandler.Delete)

		r.Get("/search/users", searchHandler.Users)
	})

	return r
}
