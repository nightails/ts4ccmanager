-- name: InsertMod :execresult
INSERT INTO mods (name, file_name, file_path, installed_files, file_hash, version, author, installed_at, status)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetModByID :one
SELECT id, name, file_name, file_path, installed_files, file_hash, version, author, installed_at, status
FROM mods
WHERE id = ?
LIMIT 1;

-- name: GetModByName :one
SELECT id, name, file_name, file_path, installed_files, file_hash, version, author, installed_at, status
FROM mods
WHERE name = ?
LIMIT 1;

-- name: GetModByPath :one
SELECT id, name, file_name, file_path, installed_files, file_hash, version, author, installed_at, status
FROM mods
WHERE file_path = ?
LIMIT 1;

-- name: GetModByHash :one
SELECT id, name, file_name, file_path, installed_files, file_hash, version, author, installed_at, status
FROM mods
WHERE file_hash = ?
LIMIT 1;

-- name: ListMods :many
SELECT id, name, file_name, file_path, installed_files, file_hash, version, author, installed_at, status
FROM mods
ORDER BY id ASC;

-- name: UpdateMod :execresult
UPDATE mods
SET name = ?, file_name = ?, file_path = ?, installed_files = ?, file_hash = ?, version = ?, author = ?, status = ?
WHERE id = ?;

-- name: DeleteMod :execresult
DELETE FROM mods
WHERE id = ?;