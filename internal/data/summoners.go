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

type Summoner struct {
	ID                        int64           `json:"id"`
	Username                  string          `json:"username"`
	Region                    string          `json:"region"`
	Rating                    int             `json:"rating"`
	CountOfPlayedGames        int             `json:"countOfPlayedGames"`
	WinRate                   float64         `json:"winRate"`
	FrequentlyPlayedChampions []ChampionStats `json:"-"`
	MatchHistory              []*Match        `json:"-"`
	AverageKDA                KDA             `json:"average_kda"`
	FrequentlyPlayedRoles     []RoleStats     `json:"-"`
}

type ChampionStats struct {
	Champion             Champion // Champion information
	CountOfPlayedMatches int      // Count of matches played with the champion
	WinRate              float64  // Winrate with the champion
}

type RoleStats struct {
	Role                 string  // Role name (e.g., "Top", "Jungle")
	CountOfPlayedMatches int     // Count of matches played in this role
	WinRate              float64 // Winrate in this role
}

type KDA struct {
	Kills   int
	Deaths  int
	Assists int
}

func ValidateSummoner(v *validator.Validator, summoner *Summoner) {
	v.Check(summoner.Username != "", "username", "must be provided")
	v.Check(summoner.Region != "", "region", "must be provided")
}

type SummonerModel struct {
	DB *sql.DB
}

func (m SummonerModel) Insert(summoner *Summoner) error {
	query := `
        INSERT INTO summoners (username, region, rating, count_of_played_games, win_rate, average_kda)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id
    `

	// Define default average KDA
	averageKDA := KDA{Kills: 0, Deaths: 0, Assists: 0}
	averageKDAJSON, err := json.Marshal(averageKDA)
	if err != nil {
		return fmt.Errorf("Insert: failed to marshal averageKDA: %v", err)
	}

	// Execute the insert query
	err = m.DB.QueryRow(query, summoner.Username, summoner.Region, 0, 0, 0, averageKDAJSON).Scan(&summoner.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("Insert: no rows were returned by the query")
		}
		return fmt.Errorf("Insert: %v", err)
	}

	return nil
}

func (k *KDA) Scan(value interface{}) error {
	byteValue, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(byteValue, k)
}

// Value implements the driver.Valuer interface for KDA.
func (k KDA) Value() (driver.Value, error) {
	return json.Marshal(k)
}

func (m SummonerModel) Get(id int64) (*Summoner, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT id, username, region, rating, count_of_played_games, win_rate, average_kda
		FROM summoners
		WHERE id = $1
	`

	var summoner Summoner

	err := m.DB.QueryRow(query, id).Scan(
		&summoner.ID,
		&summoner.Username,
		&summoner.Region,
		&summoner.Rating,
		&summoner.CountOfPlayedGames,
		&summoner.WinRate,
		&summoner.AverageKDA,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, fmt.Errorf("Get: %v", err)
		}
	}

	return &summoner, nil
}

func (m SummonerModel) Update(summoner *Summoner) error {
	query := `
		UPDATE summoners
		SET username = $1, region = $2, rating = $3, count_of_played_games = $4, win_rate = $5, average_kda = $6
		WHERE id = $7
	`

	args := []interface{}{
		summoner.Username,
		summoner.Region,
		summoner.Rating,
		summoner.CountOfPlayedGames,
		summoner.WinRate,
		summoner.AverageKDA,
		summoner.ID,
	}

	return m.DB.QueryRow(query, args...).Err()
}

func (m SummonerModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
		DELETE FROM summoners
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

func (m SummonerModel) GetAll(username string, region string, filters Filters) ([]*Summoner, error) {
	query := fmt.Sprintf(`
        SELECT id, username, region, rating, count_of_played_games, win_rate, average_kda
        FROM summoners
        WHERE (LOWER(username) = LOWER($1) OR $1 = '')
        AND (LOWER(region) = LOWER($2) OR $2 = '')
        ORDER BY %s %s, id ASC
        LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, username, region, filters.limit(), filters.offset())
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	summoners := []*Summoner{}

	for rows.Next() {
		var summoner Summoner
		err := rows.Scan(
			&summoner.ID,
			&summoner.Username,
			&summoner.Region,
			&summoner.Rating,
			&summoner.CountOfPlayedGames,
			&summoner.WinRate,
			&summoner.AverageKDA,
		)
		if err != nil {
			return nil, err
		}
		summoners = append(summoners, &summoner)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return summoners, nil
}


