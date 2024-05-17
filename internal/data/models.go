package data

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Define a custom ErrRecordNotFound error. We'll return this from our Get() method when
// looking up a movie that doesn't exist in our database.
var (
	ErrRecordNotFound = errors.New("record not found")

	ErrEditConflict = errors.New("edit conflict")
)

// Create a Models struct which wraps the MovieModel. We'll add other models to this,
// like a UserModel and PermissionModel, as our build progresses.
type Models struct {
	Champions   ChampionModel
	Matches     MatchModel
	Summoners   SummonerModel
	Users       UserModel
	Tokens      TokenModel
	Permissions PermissionModel
}

// For ease of use, we also add a New() method which returns a Models struct containing
// the initialized MovieModel.
func NewModels(db *sql.DB) Models {
	return Models{
		Champions:   ChampionModel{DB: db},
		Matches:     MatchModel{DB: db},
		Summoners:   SummonerModel{DB: db},
		Users:       UserModel{DB: db},
		Tokens:      TokenModel{DB: db},
		Permissions: PermissionModel{DB: db},
	}
}

func (m *MatchModel) UpdateSummonerStatistics(summonerID int64, champion Champion, kda KDA, role string, won bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update summoner's count of played games and win rate
	var playedGames int
	var wins int
	var avgKDA KDA
	err = tx.QueryRowContext(ctx, `
        SELECT count_of_played_games, win_rate, average_kda
        FROM summoners
        WHERE id = $1
    `, summonerID).Scan(&playedGames, &wins, &avgKDA)
	if err != nil {
		return err
	}

	playedGames++
	if won {
		wins++
	}
	winRate := float64(wins) / float64(playedGames)

	// Update average KDA
	avgKDA.Kills += kda.Kills
	avgKDA.Deaths += kda.Deaths
	avgKDA.Assists += kda.Assists
	averageKDA := KDA{
		Kills:   avgKDA.Kills / playedGames,
		Deaths:  avgKDA.Deaths / playedGames,
		Assists: avgKDA.Assists / playedGames,
	}

	// Update summoner's frequently played champions
	var championStats ChampionStats
	err = tx.QueryRowContext(ctx, `
        SELECT count_of_played_matches, win_rate
        FROM summoner_champion_stats
        WHERE summoner_id = $1 AND champion_id = $2
    `, summonerID, champion.ID).Scan(&championStats.CountOfPlayedMatches, &championStats.WinRate)
	if err == sql.ErrNoRows {
		championStats.CountOfPlayedMatches = 0
		championStats.WinRate = 0
	} else if err != nil {
		return err
	}

	championStats.CountOfPlayedMatches++
	if won {
		championStats.WinRate = float64(championStats.WinRate*float64(championStats.CountOfPlayedMatches-1)+1) / float64(championStats.CountOfPlayedMatches)
	} else {
		championStats.WinRate = float64(championStats.WinRate*float64(championStats.CountOfPlayedMatches-1)) / float64(championStats.CountOfPlayedMatches)
	}

	// Upsert the champion stats
	_, err = tx.ExecContext(ctx, `
        INSERT INTO summoner_champion_stats (summoner_id, champion_id, count_of_played_matches, win_rate)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (summoner_id, champion_id) DO UPDATE
        SET count_of_played_matches = $3, win_rate = $4
    `, summonerID, champion.ID, championStats.CountOfPlayedMatches, championStats.WinRate)
	if err != nil {
		return err
	}

	// Update summoner's frequently played roles
	var roleStats RoleStats
	err = tx.QueryRowContext(ctx, `
        SELECT count_of_played_matches, win_rate
        FROM summoner_role_stats
        WHERE summoner_id = $1 AND role = $2
    `, summonerID, role).Scan(&roleStats.CountOfPlayedMatches, &roleStats.WinRate)
	if err == sql.ErrNoRows {
		roleStats.CountOfPlayedMatches = 0
		roleStats.WinRate = 0
	} else if err != nil {
		return err
	}

	roleStats.CountOfPlayedMatches++
	if won {
		roleStats.WinRate = float64(roleStats.WinRate*float64(roleStats.CountOfPlayedMatches-1)+1) / float64(roleStats.CountOfPlayedMatches)
	} else {
		roleStats.WinRate = float64(roleStats.WinRate*float64(roleStats.CountOfPlayedMatches-1)) / float64(roleStats.CountOfPlayedMatches)
	}

	// Upsert the role stats
	_, err = tx.ExecContext(ctx, `
        INSERT INTO summoner_role_stats (summoner_id, role, count_of_played_matches, win_rate)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (summoner_id, role) DO UPDATE
        SET count_of_played_matches = $3, win_rate = $4
    `, summonerID, role, roleStats.CountOfPlayedMatches, roleStats.WinRate)
	if err != nil {
		return err
	}

	// Update the summoner's overall statistics
	_, err = tx.ExecContext(ctx, `
        UPDATE summoners
        SET count_of_played_games = $1, win_rate = $2, average_kda = $3
        WHERE id = $4
    `, playedGames, winRate, averageKDA, summonerID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// UpdateChampionStatistics updates the statistics of a champion based on the match result.
func (m *MatchModel) UpdateChampionStatistics(championID int64, summonerID int64, won bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update champion's match history and win rate
	var matchHistoryCount int
	var wins int
	err = tx.QueryRowContext(ctx, `
        SELECT count_of_played_matches, win_rate
        FROM champions
        WHERE id = $1
    `, championID).Scan(&matchHistoryCount, &wins)
	if err != nil {
		return err
	}

	matchHistoryCount++
	if won {
		wins++
	}
	winRate := float64(wins) / float64(matchHistoryCount)

	// Update champion's popularity
	var summonerCount int
	err = tx.QueryRowContext(ctx, `
        SELECT COUNT(DISTINCT summoner_id)
        FROM summoner_champion_stats
        WHERE champion_id = $1
    `, championID).Scan(&summonerCount)
	if err != nil {
		return err
	}
	popularity := float64(summonerCount)

	// Update the champion's overall statistics
	_, err = tx.ExecContext(ctx, `
        UPDATE champions
        SET count_of_played_matches = $1, win_rate = $2, popularity = $3
        WHERE id = $4
    `, matchHistoryCount, winRate, popularity, championID)
	if err != nil {
		return err
	}

	// Update best summoners
	var summonerStats SummonerChampionStats
	err = tx.QueryRowContext(ctx, `
        SELECT win_rate, count_of_played_matches
        FROM summoner_champion_stats
        WHERE summoner_id = $1 AND champion_id = $2
    `, summonerID, championID).Scan(&summonerStats.WinRate, &summonerStats.CountOfPlayedMatches)
	if err != nil {
		return err
	}

	// Upsert the best summoners stats
	_, err = tx.ExecContext(ctx, `
        INSERT INTO champion_best_summoners (champion_id, summoner_id, win_rate, count_of_played_matches)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (champion_id, summoner_id) DO UPDATE
        SET win_rate = $3, count_of_played_matches = $4
    `, championID, summonerID, summonerStats.WinRate, summonerStats.CountOfPlayedMatches)
	if err != nil {
		return err
	}

	return tx.Commit()
}

