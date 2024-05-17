package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"league_of_graphs.satellite.net/internal/validator"
)

type Champion struct {
	ID            int64                   `json:"id"`
	Name          string                  `json:"name"`
	MainRole      string                  `json:"mainRole"`
	Popularity    float64                 `json:"popularity"`
	WinRate       float64                 `json:"winRate"`
	BanRate       float64                 `json:"banRate"`
	MatchHistory  []*Match                `json:"-"`
	BestSummoners []SummonerChampionStats `json:"-"`
}

type SummonerChampionStats struct {
	Summoner             Summoner // Summoner information
	WinRate              float64  // Winrate with the champion
	CountOfPlayedMatches int      // Count of matches played with the champion
}

func ValidateChampion(v *validator.Validator, champion *Champion) {
	v.Check(champion.Name != "", "name", "must be provided")
	v.Check(champion.MainRole != "", "main_role", "must be provided")

	v.Check(champion.Name != "Champion", "name", "must be different from the name of the champion")
}

type ChampionModel struct {
	DB *sql.DB
}

func (m ChampionModel) Insert(champion *Champion) error {
	query := `
        INSERT INTO champions (name, main_role)
        VALUES ($1, $2)
        RETURNING id, popularity, win_rate, ban_rate
    `

	args := []interface{}{champion.Name, champion.MainRole}

	return m.DB.QueryRow(query, args...).Scan(&champion.ID, &champion.Popularity, &champion.WinRate, &champion.BanRate)
}

func (c ChampionModel) Get(id int64) (*Champion, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT id, name, main_role, popularity, win_rate, ban_rate
		FROM champions
		WHERE id = $1
	`

	var champion Champion

	err := c.DB.QueryRow(query, id).Scan(
		&champion.ID,
		&champion.Name,
		&champion.MainRole,
		&champion.Popularity,
		&champion.WinRate,
		&champion.BanRate,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &champion, nil
}

func (c ChampionModel) Update(champion *Champion) error {
	query := `
		UPDATE champions
		SET name = $1, main_role = $2
		WHERE id = $3
	`

	args := []interface{}{champion.Name, champion.MainRole, champion.ID}

	return c.DB.QueryRow(query, args...).Err()
}

func (c ChampionModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
		DELETE FROM champions
		WHERE id = $1
	`
	result, err := c.DB.Exec(query, id)
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

func (f Filters) limit() int {
	return f.PageSize
}

func (f Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}

func (c ChampionModel) GetAll(username string, region string, filters Filters) ([]*Champion, error) {
	query := fmt.Sprintf(`
        SELECT id, name, main_role, popularity, win_rate, ban_rate
        FROM champions
        WHERE (LOWER(name) = LOWER($1) OR $1 = '')
        AND (LOWER(main_role) = LOWER($2) OR $2 = '')
        ORDER BY %s %s, id ASC
        LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := c.DB.QueryContext(ctx, query, username, region, filters.limit(), filters.offset())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	champions := []*Champion{}

	for rows.Next() {
		var champion Champion
		err := rows.Scan(
			&champion.ID,
			&champion.Name,
			&champion.MainRole,
			&champion.Popularity,
			&champion.WinRate,
			&champion.BanRate,
		)
		if err != nil {
			return nil, err
		}
		champions = append(champions, &champion)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return champions, nil
}
