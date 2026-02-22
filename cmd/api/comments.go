package main

import (
	"errors"
	"net/http"

	"github.com/manas-solves/realworld-backend/internal/data"
	"github.com/manas-solves/realworld-backend/internal/validator"
	"github.com/go-chi/chi/v5"
)

func (app *application) createCommentHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var input struct {
		Comment struct {
			Body string `json:"body"`
		} `json:"comment"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Get the article ID by slug
	articleID, err := app.modelStore.Articles.GetIDBySlug(slug)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	comment := &data.Comment{
		Body:      input.Comment.Body,
		ArticleID: articleID,
		AuthorID:  app.contextGetUser(r).ID,
	}

	v := validator.New()

	if data.ValidateComment(v, comment); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	currentUser := app.contextGetUser(r)

	// Insert comment and get complete comment with author in a single operation
	// Uses currentUser from context instead of querying database
	createdComment, err := app.modelStore.Comments.InsertAndReturn(comment, currentUser)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"comment": createdComment}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) getCommentsHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	// Get the article ID by slug (verifies article exists)
	articleID, err := app.modelStore.Articles.GetIDBySlug(slug)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	// Get all comments for the article (includes author details via JOIN)
	comments, err := app.modelStore.Comments.GetByArticleID(articleID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Set following status if user is authenticated (single bulk query)
	currentUser := app.contextGetUser(r)
	if !currentUser.IsAnonymous() {
		err = app.modelStore.Comments.SetFollowingStatus(comments, currentUser.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"comments": comments}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}
