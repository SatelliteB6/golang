package data

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"league_of_graphs.satellite.net/internal/validator"
)

type Match struct {
	ID         int64     `json:"id"`
	PlayedDate time.Time `json:"playedDate"`
	Duration   int       `json:"duration"`
	Result     string    `json:"result"`
	BlueTeam   *Team     `json:"blueTeam"`
	RedTeam    *Team     `json:"redTeam"`
}

type Team struct {
	TeamKDA             KDA                         // Team's total KDA
	TurretsDestroyed    int                         // Number of turrets destroyed
	InhibitorsDestroyed int                         // Number of inhibitors destroyed
	RiftHeraldsKilled   int                         // Number of Rift Heralds killed
	DragonsKilled       int                         // Number of dragons killed
	BaronNashorsKilled  int                         // Number of Baron Nashors killed
	Summoners           []*SummonerMatchPerformance // List of summoners in the team
	BannedChampions     []Champion                  // List of banned champions
}

type SummonerMatchPerformance struct {
	Username    string       // Summoner information
	Champion    ChampionData // Champion played by the summoner
	NetWorth    int          // Net worth of the summoner in the match
	KDA         KDA          // KDA of the summoner in the match
	BoughtItems []string     // List of items bought by the summoner
}

type ChampionData struct {
	Name     string `json:"name"`
	MainRole string `json:"mainRole"`
}

func ValidateMatch(v *validator.Validator, match *Match) {
	v.Check(match.Result != "", "result", "must be provided")
	v.Check(match.Duration > 0, "duration", "must be provided")
	v.Check(match.BlueTeam != nil, "blue_team", "must be provided")
	v.Check(match.RedTeam != nil, "red_team", "must be provided")
}

type MatchModel struct {
	DB *sql.DB
}

func (m MatchModel) Insert(match *Match) error {
	query := `
        INSERT INTO matches (duration, result, played_date, blue_team, red_team)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `

	blueTeamJSON, err := json.Marshal(match.BlueTeam)
	if err != nil {
		return fmt.Errorf("Insert: %v", err)
	}

	redTeamJSON, err := json.Marshal(match.RedTeam)
	if err != nil {
		return fmt.Errorf("Insert: %v", err)
	}

	args := []interface{}{match.Duration, match.Result, match.PlayedDate, blueTeamJSON, redTeamJSON}

	return m.DB.QueryRow(query, args...).Scan(&match.ID)
}

func (t *Team) Scan(value interface{}) error {
	byteValue, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(byteValue, t)
}

// Value implements the driver.Valuer interface for Team.
func (t Team) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (m MatchModel) Get(id int64) (*Match, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT id, duration, result, played_date, blue_team, red_team
		FROM matches
		WHERE id = $1
	`

	var match Match

	err := m.DB.QueryRow(query, id).Scan(
		&match.ID,
		&match.Duration,
		&match.Result,
		&match.PlayedDate,
		&match.BlueTeam,
		&match.RedTeam,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &match, nil
}

func (m MatchModel) Update(match *Match) error {
	query := `
		UPDATE matches
		SET duration = $1, result = $2, played_date = $3, blue_team = $4, red_team = $5
		WHERE id = $6
	`

	args := []interface{}{
		match.Duration,
		match.Result,
		match.PlayedDate,
		match.BlueTeam,
		match.RedTeam,
		match.ID,
	}

	return m.DB.QueryRow(query, args...).Err()
}

func (m MatchModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
		DELETE FROM matches
		WHERE id = $1
	`

	result, err := m.DB.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (m MatchModel) GetAll(filters Filters) ([]*Match, error) {
	query := fmt.Sprintf(`
        SELECT id, duration, result, played_date, blue_team, red_team
        FROM matches
        ORDER BY %s %s, id ASC
        LIMIT $1 OFFSET $2`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, filters.limit(), filters.offset())
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	matches := []*Match{}

	for rows.Next() {
		var match Match
		err := rows.Scan(
			&match.ID,
			&match.Duration,
			&match.Result,
			&match.PlayedDate,
			&match.BlueTeam,
			&match.RedTeam,
		)
		if err != nil {
			return nil, err
		}
		matches = append(matches, &match)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return matches, nil
}
