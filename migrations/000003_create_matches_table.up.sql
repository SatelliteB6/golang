CREATE TABLE IF NOT EXISTS matches (
    id bigserial PRIMARY KEY,
    played_date timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    duration integer NOT NULL,
    result text NOT NULL,
    blue_team jsonb NOT NULL,
    red_team jsonb NOT NULL
);
