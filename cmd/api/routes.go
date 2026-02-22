package main

import (
	"github.com/go-chi/chi/v5"
)

// routes returns a new chi router containing the application routes.
func (app *application) routes() *chi.Mux {
	r := chi.NewRouter()

	r.NotFound(app.notFoundResponse)
	r.MethodNotAllowed(app.methodNotAllowedResponse)

	r.Use(app.recoverPanic, app.authenticate)

	r.Get("/healthcheck", app.healthcheckHandler)

	r.Route("/users", func(r chi.Router) {
		r.Post("/", app.registerUserHandler)
		r.Post("/login", app.loginUserHandler)
	})

	r.Route("/user", func(r chi.Router) {
		r.Use(app.requireAuthenticatedUser)
		r.Get("/", app.getCurrentUserHandler)
		r.Put("/", app.updateUserHandler)
	})

	r.Route("/profiles/{username}", func(r chi.Router) {
		r.Get("/", app.getProfileHandler)
		r.With(app.requireAuthenticatedUser).Post("/follow", app.followUserHandler)
		r.With(app.requireAuthenticatedUser).Delete("/follow", app.unfollowUserHandler)
	})

	r.Route("/articles", func(r chi.Router) {
		r.Get("/", app.listArticlesHandler)
		r.With(app.requireAuthenticatedUser).Get("/feed", app.feedArticlesHandler)
		r.With(app.requireAuthenticatedUser).Post("/", app.createArticleHandler)
		r.Get("/{slug}", app.getArticleHandler)
		r.With(app.requireAuthenticatedUser).Put("/{slug}", app.updateArticleHandler)
		r.With(app.requireAuthenticatedUser).Delete("/{slug}", app.deleteArticleHandler)
		r.With(app.requireAuthenticatedUser).Post("/{slug}/favorite", app.favoriteArticleHandler)
		r.With(app.requireAuthenticatedUser).Delete("/{slug}/favorite", app.unfavoriteArticleHandler)
		r.With(app.requireAuthenticatedUser).Post("/{slug}/comments", app.createCommentHandler)
		r.Get("/{slug}/comments", app.getCommentsHandler)
	})

	r.Get("/tags", app.getTagsHandler)

	return r
}
