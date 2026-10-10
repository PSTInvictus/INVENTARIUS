-- Active = not released and not past its "until" date (inclusive).

-- name: CreateReservation :one
INSERT INTO reservations (item_id, stock_id, asset_id, quantity, user_id, project_id, department_id, note, until)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetReservation :one
SELECT * FROM reservations WHERE id = ?;

-- name: ReservedOnStock :one
SELECT CAST(COALESCE(sum(quantity), 0) AS REAL) FROM reservations
WHERE stock_id = ? AND released_at IS NULL AND (until IS NULL OR until >= date('now'));

-- name: ActiveReservationForAsset :one
SELECT * FROM reservations
WHERE asset_id = ? AND released_at IS NULL AND (until IS NULL OR until >= date('now'))
ORDER BY id LIMIT 1;

-- name: ListActiveReservationsForItem :many
SELECT sqlc.embed(reservations), users.name AS user_name
FROM reservations JOIN users ON users.id = reservations.user_id
WHERE reservations.item_id = ? AND reservations.released_at IS NULL
  AND (reservations.until IS NULL OR reservations.until >= date('now'))
ORDER BY reservations.id;

-- name: SetReservationQuantity :exec
UPDATE reservations SET quantity = ? WHERE id = ?;

-- name: ReleaseReservation :exec
UPDATE reservations SET released_at = CURRENT_TIMESTAMP WHERE id = ? AND released_at IS NULL;
