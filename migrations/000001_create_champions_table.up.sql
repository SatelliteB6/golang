CREATE TABLE IF NOT EXISTS champions (
    id bigserial PRIMARY KEY,
    name text NOT NULL,
    main_role text NOT NULL,
    popularity float8 DEFAULT 0,
    win_rate float8 DEFAULT 0,
    ban_rate float8 DEFAULT 0
);

CREATE TABLE IF NOT EXISTS summoner_champion_stats (
    id bigserial PRIMARY KEY,
    summoner_id bigserial NOT NULL REFERENCES summoners(id),
    champion_id bigserial NOT NULL REFERENCES champions(id),
    win_rate float8 NOT NULL,
    count_of_played_matches integer NOT NULL
);
