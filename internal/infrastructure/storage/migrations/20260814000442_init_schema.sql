-- +goose Up
-- +goose StatementBegin
CREATE TYPE user_role AS ENUM (
    'member',
    'admin'
);

CREATE TYPE space_type AS ENUM (
    'meeting_room',
    'desk'
);


CREATE TABLE users (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role       user_role NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE TABLE spaces (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    type       space_type NOT NULL,
    capacity   INTEGER NOT NULL CHECK (capacity > 0),
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


CREATE TABLE reservations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    space_id   UUID NOT NULL
        REFERENCES spaces(id)
        ON DELETE RESTRICT,

    user_id    UUID NOT NULL
        REFERENCES users(id)
        ON DELETE RESTRICT,

    date       DATE NOT NULL,

    slot       SMALLINT NOT NULL
        CHECK (slot >= 0 AND slot < 288),

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT reservations_space_date_slot_unique
        UNIQUE (space_id, date, slot)
);


CREATE INDEX idx_reservations_user_date
    ON reservations (user_id, date, space_id, slot);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS reservations;
DROP TABLE IF EXISTS spaces;
DROP TYPE IF EXISTS space_type;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS user_role;
-- +goose StatementEnd