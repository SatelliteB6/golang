package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"league_of_graphs.satellite.net/internal/data"
	"league_of_graphs.satellite.net/internal/validator"
)

// Add a createMovieHandler for the "POST /v1/movies" endpoint. For now we simply
// return a plain-text placeholder response.
func (app *application) createSummonerHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string `json:"username"`
		Region   string `json:"region"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Copy the values from the input struct to a new Summoner struct.
	summoner := &data.Summoner{
		Username: input.Username,
		Region:   input.Region,
	}

	// Initialize a new Validator.
	v := validator.New()

	// Call the ValidateSummoner() function and return a response containing the errors if any of the checks fail.
	if data.ValidateSummoner(v, summoner); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Summoners.Insert(summoner)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/summoners/%d", summoner.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"summoner": summoner}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// Add a showMovieHandler for the "GET /v1/movies/:id" endpoint. For now, we retrieve
// the interpolated "id" parameter from the current URL and include it in a placeholder
// response.
func (app *application) showSummonerHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Create a new instance of the Summoner struct with dummy data.
	summoner, err := app.models.Summoners.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Encode the struct to JSON and send it as the HTTP response.
	err = app.writeJSON(w, http.StatusOK, envelope{"summoner": summoner}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateSummonerHandler(w http.ResponseWriter, r *http.Request) {

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	summoner, err := app.models.Summoners.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Username string `json:"username"`
		Region   string `json:"region"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	summoner.Username = input.Username
	summoner.Region = input.Region

	v := validator.New()

	if data.ValidateSummoner(v, summoner); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Summoners.Update(summoner)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"summoner": summoner}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteSummonerHandler(w http.ResponseWriter, r *http.Request) {

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Summoners.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "summoner successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listSummonersHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Username string
		Region   string
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	input.Username = app.readString(qs, "username", "")
	input.Region = app.readString(qs, "region", "")

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "username", "region", "-id", "-username", "-region"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	summoners, err := app.models.Summoners.GetAll(input.Username, input.Region, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"summoners": summoners}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

type SummonerMatchPerformance struct {
	Username      string
	Champion      ChampionData
	NetWorth      int
	KDA           KDA
	BoughtItems   []string
	MatchDuration time.Duration
	MatchDate     time.Time
	MatchResult   string
	MatchID       int
}

type ChampionData struct {
	Name     string
	MainRole string
}

type KDA struct {
	Kills   int
	Deaths  int
	Assists int
}

func (app *application) getSummonersByMatch(w http.ResponseWriter, r *http.Request) {
	params := httprouter.ParamsFromContext(r.Context())
	matchID, err := strconv.Atoi(params.ByName("id"))
	if err != nil {
		http.Error(w, "Invalid match ID", http.StatusBadRequest)
		return
	}

	query := `
        SELECT s.username, c.name, c.main_role, mp.net_worth, mp.kills, mp.deaths, mp.assists, mp.bought_items
        FROM summoners s
        JOIN match_performance mp ON s.id = mp.summoner_id
        JOIN champions c ON mp.champion_id = c.id
        WHERE mp.match_id = $1
    `

	rows, err := app.DB.Query(query, matchID)
	if err != nil {
		http.Error(w, "Query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	summoners := []SummonerMatchPerformance{}

	for rows.Next() {
		var summoner SummonerMatchPerformance
		var boughtItems string
		if err := rows.Scan(&summoner.Username, &summoner.Champion.Name, &summoner.Champion.MainRole, &summoner.NetWorth, &summoner.KDA.Kills, &summoner.KDA.Deaths, &summoner.KDA.Assists, &boughtItems); err != nil {
			http.Error(w, "Row scan error", http.StatusInternalServerError)
			return
		}
		json.Unmarshal([]byte(boughtItems), &summoner.BoughtItems)
		summoners = append(summoners, summoner)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Rows error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summoners)
}
