package main

import (
	"errors"
	"net/http"

	"github.com/manas-solves/realworld-backend/internal/data"
	"github.com/manas-solves/realworld-backend/internal/validator"
	"github.com/go-chi/chi/v5"
)

func (app *application) listArticlesHandler(w http.ResponseWriter, r *http.Request) {
	// Read pagination parameters using reusable helper
	// Default limit is 20, max limit is 100
	pagination := app.readPagination(r, 20, 100)

	// Read query parameters
	qs := r.URL.Query()

	// Read filters
	filters := data.ArticleFilters{
		Tag:       qs.Get("tag"),
		Author:    qs.Get("author"),
		Favorited: qs.Get("favorited"),
		Limit:     pagination.Limit,
		Offset:    pagination.Offset,
	}

	// Validate filters
	v := validator.New()
	filters.Validate(v)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Get current user (may be anonymous)
	currentUser := app.contextGetUser(r)

	// List articles with filters
	articles, totalCount, err := app.modelStore.Articles.List(filters, currentUser)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Write response
	err = app.writeJSON(w, http.StatusOK, envelope{
		"articles":      articles,
		"articlesCount": totalCount,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) feedArticlesHandler(w http.ResponseWriter, r *http.Request) {
	// Read pagination parameters using reusable helper
	// Default limit is 20, max limit is 100
	pagination := app.readPagination(r, 20, 100)

	// Get current user (authentication required for feed)
	currentUser := app.contextGetUser(r)

	// Create filters for feed - only get articles from followed users
	filters := data.ArticleFilters{
		Feed:   true,
		Limit:  pagination.Limit,
		Offset: pagination.Offset,
	}

	// Get articles using List method with Feed filter
	articles, totalCount, err := app.modelStore.Articles.List(filters, currentUser)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Write response
	err = app.writeJSON(w, http.StatusOK, envelope{
		"articles":      articles,
		"articlesCount": totalCount,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) createArticleHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Article struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Body        string   `json:"body"`
			TagList     []string `json:"tagList"`
		} `json:"article"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	article := &data.Article{
		Title:       input.Article.Title,
		Description: input.Article.Description,
		Body:        input.Article.Body,
		TagList:     input.Article.TagList,
		AuthorID:    app.contextGetUser(r).ID,
	}

	v := validator.New()

	if data.ValidateArticle(v, article); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Insert article and get complete article with author in a single query
	// Tags are inserted synchronously as part of the article insertion
	createdArticle, err := app.modelStore.Articles.InsertAndReturn(article, app.contextGetUser(r))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return response with created article
	headers := make(http.Header)
	headers.Set("Location", "/articles/"+createdArticle.Slug)
	err = app.writeJSON(w, http.StatusCreated, envelope{"article": createdArticle}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getArticleHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	article, err := app.modelStore.Articles.GetBySlug(slug, app.contextGetUser(r))
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"article": article}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) favoriteArticleHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := app.contextGetUser(r)

	article, err := app.modelStore.Articles.FavoriteBySlug(slug, user.ID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"article": article}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) unfavoriteArticleHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := app.contextGetUser(r)

	article, err := app.modelStore.Articles.UnfavoriteBySlug(slug, user.ID)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{"article": article}, nil); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) deleteArticleHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := app.contextGetUser(r)

	err := app.modelStore.Articles.DeleteBySlug(slug, user.ID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *application) updateArticleHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	user := app.contextGetUser(r)

	article, err := app.modelStore.Articles.GetBySlug(slug, user)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	if article.Author.Username != user.Username {
		app.notPermittedResponse(w, r)
		return
	}

	var input struct {
		Article struct {
			Title       *string `json:"title"`
			Description *string `json:"description"`
			Body        *string `json:"body"`
		} `json:"article"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Article.Title != nil {
		article.Title = *input.Article.Title
		article.GenerateSlug()
	}

	if input.Article.Description != nil {
		article.Description = *input.Article.Description
	}

	if input.Article.Body != nil {
		article.Body = *input.Article.Body
	}

	v := validator.New()
	if data.ValidateArticle(v, article); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.modelStore.Articles.Update(article)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// set location header to point to the new article
	headers := make(http.Header)
	headers.Set("Location", "/articles/"+article.Slug)
	err = app.writeJSON(w, http.StatusOK, envelope{"article": article}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
