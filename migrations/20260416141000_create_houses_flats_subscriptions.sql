-- +goose Up
-- +goose StatementBegin
CREATE TABLE houses (
    id BIGINT PRIMARY KEY,
    address TEXT NOT NULL,
    year INT NOT NULL,
    developer TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_flat_created_at TIMESTAMPTZ
);

CREATE TABLE flats (
    id BIGSERIAL PRIMARY KEY,
    house_id BIGINT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
    number INT NOT NULL,
    price BIGINT NOT NULL,
    rooms INT NOT NULL,
    status TEXT NOT NULL,
    created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    moderation_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (house_id, number)
);

CREATE TABLE subscriptions (
    id BIGSERIAL PRIMARY KEY,
    house_id BIGINT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (house_id, email)
);

CREATE INDEX flats_house_id_idx ON flats (house_id, created_at DESC);
CREATE INDEX flats_house_status_idx ON flats (house_id, status, created_at DESC);
CREATE INDEX subscriptions_house_id_idx ON subscriptions (house_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS flats;
DROP TABLE IF EXISTS houses;
-- +goose StatementEnd
