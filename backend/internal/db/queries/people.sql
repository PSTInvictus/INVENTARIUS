-- name: GetUser :one
SELECT * FROM users WHERE id = ?;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: CreateUser :one
INSERT INTO users (email, name, picture, role) VALUES (?, ?, ?, ?) RETURNING *;

-- name: UpdateUserProfile :exec
UPDATE users SET name = ?, picture = ? WHERE id = ?;

-- name: UpdateUserAccess :exec
UPDATE users SET role = ?, active = ? WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY name;

-- name: ListUserDepartmentIDs :many
SELECT department_id FROM user_departments WHERE user_id = ? ORDER BY department_id;

-- name: ListAllUserDepartments :many
SELECT * FROM user_departments ORDER BY user_id, department_id;

-- name: ClearUserDepartments :exec
DELETE FROM user_departments WHERE user_id = ?;

-- name: AddUserDepartment :exec
INSERT INTO user_departments (user_id, department_id) VALUES (?, ?);

-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?);

-- name: GetSessionUser :one
SELECT sqlc.embed(users) FROM sessions
JOIN users ON users.id = sessions.user_id
WHERE sessions.token_hash = ? AND sessions.expires_at > ? AND users.active = 1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= ?;

-- name: ListProjects :many
SELECT * FROM projects ORDER BY id;

-- name: ListDepartments :many
SELECT * FROM departments ORDER BY project_id, id;

-- name: GetDepartment :one
SELECT * FROM departments WHERE id = ?;
