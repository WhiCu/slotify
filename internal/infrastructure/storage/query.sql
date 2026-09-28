-- ============================================================
-- Users
-- ============================================================

-- name: CountUsers :one
SELECT count(*)::bigint
FROM users;


-- name: LockRootCreation :exec
SELECT pg_advisory_xact_lock(
    hashtextextended('slotify:create-root-user', 0)
);


-- name: SaveUser :exec
INSERT INTO users (
    id,
    role,
    created_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(role)::user_role,
    sqlc.arg(created_at)
)
ON CONFLICT (id) DO UPDATE
SET
    role = EXCLUDED.role;


-- name: FindUserByID :one
SELECT *
FROM users
WHERE id = sqlc.arg(id);


-- name: FindUserByIDForUpdate :one
SELECT *
FROM users
WHERE id = sqlc.arg(id)
FOR UPDATE;


-- name: ListUsers :many
SELECT *
FROM users
ORDER BY created_at, id;


-- name: DeleteUser :exec
DELETE FROM users
WHERE id = sqlc.arg(id);


-- ============================================================
-- Spaces
-- ============================================================

-- name: SaveSpace :exec
INSERT INTO spaces (
    id,
    name,
    type,
    capacity,
    active,
    created_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(name),
    sqlc.arg(type)::space_type,
    sqlc.arg(capacity),
    sqlc.arg(active),
    sqlc.arg(created_at)
)
ON CONFLICT (id) DO UPDATE
SET
    name = EXCLUDED.name,
    type = EXCLUDED.type,
    capacity = EXCLUDED.capacity,
    active = EXCLUDED.active;


-- name: FindSpaceByID :one
SELECT *
FROM spaces
WHERE id = sqlc.arg(id);


-- name: FindSpaceByIDForUpdate :one
SELECT *
FROM spaces
WHERE id = sqlc.arg(id)
FOR UPDATE;


-- name: ListSpaces :many
SELECT *
FROM spaces
WHERE
    NOT sqlc.arg(only_active)::boolean
    OR active
ORDER BY name, id;


-- name: DeleteSpace :exec
DELETE FROM spaces
WHERE id = sqlc.arg(id);


-- ============================================================
-- Reservations
-- ============================================================

-- name: SaveReservations :batchexec
INSERT INTO reservations (
    id,
    space_id,
    user_id,
    date,
    slot,
    created_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
);


-- name: FindReservationByID :one
SELECT *
FROM reservations
WHERE id = sqlc.arg(id);


-- name: FindReservationByIDForUpdate :one
SELECT *
FROM reservations
WHERE id = sqlc.arg(id)
FOR UPDATE;


-- name: HasReservationConflict :one
SELECT EXISTS (
    SELECT 1
    FROM reservations
    WHERE space_id = sqlc.arg(space_id)
      AND date = sqlc.arg(date)
      AND slot = ANY(sqlc.arg(slots)::smallint[])
) AS conflict;


-- name: ListReservationsBySpaceAndDate :many
SELECT *
FROM reservations
WHERE space_id = sqlc.arg(space_id)
  AND date = sqlc.arg(date)
ORDER BY slot;


-- name: ListReservationsByUserID :many
SELECT *
FROM reservations
WHERE user_id = sqlc.arg(user_id)
ORDER BY date, space_id, slot;


-- name: DeleteReservationsByIDs :exec
DELETE FROM reservations
WHERE id = ANY(sqlc.arg(ids)::uuid[]);


-- name: DeleteReservationsByUserID :exec
DELETE FROM reservations
WHERE user_id = sqlc.arg(user_id);