package main

import (
	"errors"
	"net/http"

	"github.com/manas-solves/realworld-backend/internal/data"
	"github.com/manas-solves/realworld-backend/internal/validator"
	"github.com/go-chi/chi/v5"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		User struct {
			Username          string `json:"username"`
			Email             string `json:"email"`
			PasswordPlaintext string `json:"password"`
		} `json:"user"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := data.User{
		Username: input.User.Username,
		Email:    input.User.Email,
	}

	err = user.Password.Set(input.User.PasswordPlaintext)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.modelStore.Users.Insert(&user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		case errors.Is(err, data.ErrDuplicateUsername):
			v.AddError("a user with this username already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	token, err := app.jwtMaker.CreateToken(user.ID, app.config.jwtMaker.accessDuration)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	user.Token = token

	err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) loginUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		User struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		} `json:"user"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Validate the email and password fields.
	v := validator.New()
	data.ValidateEmail(v, input.User.Email)
	data.ValidatePasswordPlaintext(v, input.User.Password)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	user, err := app.modelStore.Users.GetByEmail(input.User.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	matches, err := user.Password.Matches(input.User.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	if !matches {
		app.invalidCredentialsResponse(w, r)
		return
	}

	// Generate a new JWT token for the user.
	token, err := app.jwtMaker.CreateToken(user.ID, app.config.jwtMaker.accessDuration)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	user.Token = token

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getCurrentUserHandler returns the currently authenticated user.
func (app *application) getCurrentUserHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
	err := app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// getProfileHandler returns a user's profile, including follow status.
func (app *application) getProfileHandler(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	targetUser, err := app.modelStore.Users.GetByUsername(username)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	var following bool
	user := app.contextGetUser(r)
	if !user.IsAnonymous() {
		following, _ = app.modelStore.Users.IsFollowing(user.ID, targetUser.ID)
	}

	profile := targetUser.ToProfile(following)
	err = app.writeJSON(w, http.StatusOK, envelope{"profile": profile}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// followUserHandler lets the authenticated user follow another user.
func (app *application) followUserHandler(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	targetUser, err := app.modelStore.Users.GetByUsername(username)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}
	user := app.contextGetUser(r)
	if user.ID == targetUser.ID {
		app.failedValidationResponse(w, r, []string{"cannot follow yourself"})
		return
	}
	err = app.modelStore.Users.FollowUser(user.ID, targetUser.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	profile := targetUser.ToProfile(true)
	err = app.writeJSON(w, http.StatusOK, envelope{"profile": profile}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// unfollowUserHandler lets the authenticated user unfollow another user.
func (app *application) unfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	targetUser, err := app.modelStore.Users.GetByUsername(username)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}
	user := app.contextGetUser(r)
	err = app.modelStore.Users.UnfollowUser(user.ID, targetUser.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	profile := targetUser.ToProfile(false)
	err = app.writeJSON(w, http.StatusOK, envelope{"profile": profile}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	var input struct {
		User struct {
			Email    *string `json:"email"`
			Password *string `json:"password"`
			Username *string `json:"username"`
			Bio      *string `json:"bio"`
			Image    *string `json:"image"`
		} `json:"user"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	updatedUser := *user
	if input.User.Email != nil {
		updatedUser.Email = *input.User.Email
	}
	if input.User.Username != nil {
		updatedUser.Username = *input.User.Username
	}
	if input.User.Bio != nil {
		updatedUser.Bio = *input.User.Bio
	}
	if input.User.Image != nil {
		updatedUser.Image = *input.User.Image
	}
	if input.User.Password != nil {
		err := updatedUser.Password.Set(*input.User.Password)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	v := validator.New()
	if data.ValidateUser(v, updatedUser); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.modelStore.Users.Update(&updatedUser)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		case errors.Is(err, data.ErrDuplicateUsername):
			v.AddError("a user with this username already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Cache invalidation is now handled automatically in UserStore.Update

	token, err := app.jwtMaker.CreateToken(user.ID, app.config.jwtMaker.accessDuration)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	updatedUser.Token = token

	err = app.writeJSON(w, http.StatusOK, envelope{"user": updatedUser}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
