CREATE TABLE IF NOT EXISTS summoners (
    id bigserial PRIMARY KEY,
    username text NOT NULL,
    region text NOT NULL,
    rating integer DEFAULT 0,
    count_of_played_games integer DEFAULT 0,
    win_rate float8 DEFAULT 0,
    average_kda jsonb NOT NULL DEFAULT '{"kills":0,"deaths":0,"assists":0}'
);

CREATE TABLE IF NOT EXISTS champion_stats (
    id bigserial PRIMARY KEY,
    summoner_id bigint NOT NULL REFERENCES summoners(id),
    champion_id bigint NOT NULL REFERENCES champions(id),
    count_of_played_matches integer DEFAULT 0,
    win_rate float8 DEFAULT 0
);

CREATE TABLE IF NOT EXISTS role_stats (
    id bigserial PRIMARY KEY,
    summoner_id bigint NOT NULL REFERENCES summoners(id),
    role text NOT NULL,
    count_of_played_matches integer DEFAULT 0,
    win_rate float8 DEFAULT 0
);
