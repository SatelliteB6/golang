package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	// Initialize a new httprouter router instance.
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodPost, "/v1/summoners", app.createSummonerHandler)
	router.HandlerFunc(http.MethodGet, "/v1/summoners/:id", app.showSummonerHandler)
	router.HandlerFunc(http.MethodPost, "/v1/matches", app.createMatchHandler)
	router.HandlerFunc(http.MethodGet, "/v1/matches/:id", app.showMatchHandler)
	router.HandlerFunc(http.MethodPost, "/v1/champions", app.createChampionHandler)
	router.HandlerFunc(http.MethodGet, "/v1/champions/:id", app.showChampionHandler)
	router.HandlerFunc(http.MethodPut, "/v1/champions/:id", app.updateChampionHandler)
	router.HandlerFunc(http.MethodPut, "/v1/summoners/:id", app.updateSummonerHandler)
	router.HandlerFunc(http.MethodPut, "/v1/matches/:id", app.updateMatchHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/summoners/:id", app.deleteSummonerHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/matches/:id", app.deleteMatchHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/champions/:id", app.deleteChampionHandler)
	router.HandlerFunc(http.MethodGet, "/v1/matches", app.listMatchesHandler)
	router.HandlerFunc(http.MethodGet, "/v1/champions", app.listChampionsHandler)
	router.HandlerFunc(http.MethodGet, "/v1/summoners", app.listSummonersHandler)

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)

	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)
	// Return the httprouter instance.

	router.HandlerFunc(http.MethodGet, "/v1/matches/:id/summoners", app.getSummonersByMatch)

	return app.metrics(app.recoverPanic(app.enableCORS(app.rateLimit(app.authenticate(router)))))
}
