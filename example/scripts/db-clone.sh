#!/bin/sh
# scripts/db-clone.sh - clone the main DB's Docker volume into this worktree
#
# Invoked from the `migration` profile's `hooks.add` (see ../zgt.config.yaml).
# Relies on env vars that `zgt env` (or the hook runner) exports:
#
#   DB_SOURCE_PROJECT   e.g. "<repo>-main"   - the shared DB's compose project
#   COMPOSE_PROJECT_NAME e.g. "<repo>-migration-x"  - this worktree's project
#
# Docker Compose names volumes as "<project>_<volume_name>" by default, so:
#   - source volume: "${DB_SOURCE_PROJECT}_pgdata"
#   - dest   volume: "${COMPOSE_PROJECT_NAME}_pgdata"
#
# We copy with `cp -a` (preserving perms/ownership) and remove `postmaster.pid`
# so the new postgres container starts up cleanly via crash recovery.
#
# NOTE: This script reads the running -- or stopped -- main DB. Copying a live
# PGDATA with `cp -a` and then starting a new container on it is the
# well-known trick for cloning a Postgres data dir; postgres's crash recovery
# replays WAL on container start. If you can stop the main DB during the copy
# you get a CLI-clean clone; otherwise prefer `pg_dump`/`pg_restore` (slower
# but safe under concurrency).
#
# Resilience: refuses to proceed if either volume does not exist.

set -eu

: "${DB_SOURCE_PROJECT:?DB_SOURCE_PROJECT must be set (e.g. repo-main)}"
: "${COMPOSE_PROJECT_NAME:?COMPOSE_PROJECT_NAME must be set to this worktree project}"

src="${DB_SOURCE_PROJECT}_pgdata"
dst="${COMPOSE_PROJECT_NAME}_pgdata"

# Sanity check: refuse to clobber an existing destination volume.
if docker volume inspect "$dst" >/dev/null 2>&1; then
  printf 'db-clone: destination volume %s already exists; refusing to overwrite.\n' "$dst" >&2
  printf 'db-clone: run `docker volume rm %s` first (e.g. via `zgt rm`).\n' "$dst" >&2
  exit 1
fi

if ! docker volume inspect "$src" >/dev/null 2>&1; then
  printf 'db-clone: source volume %s does not exist; bring up the main DB first.\n' "$src" >&2
  exit 1
fi

# Create the destination volume up front so it has the correct driver/labels.
docker volume create "$dst" >/dev/null

# Copy the data with alpine and strip the stale postmaster pid so the new
# postgres container does not complain about a running postmaster.
docker run --rm \
  -v "${src}:/src:ro" \
  -v "${dst}:/dst" \
  alpine:3 sh -s <<'SH'
set -eu
cp -a /src/. /dst/
rm -f /dst/postmaster.pid
SH

printf 'db-clone: cloned %s -> %s\n' "$src" "$dst"