# Example: per-worktree Docker Compose with isolated DB

This directory illustrates a complete `zgt` profiles setup for the
"DB stays on the main worktree, api/envoy fork per branch, and a migration
profile clones the DB volume when you need an isolated DB" pattern.

## Layout

```
example/
├── docker-compose.yaml   # compose services for db / api / envoy
├── envoy.yaml             # minimal envoy config referenced by compose
├── zgt.config.yaml        # zgt config with a `migration` profile
└── scripts/
    └── db-clone.sh        # volume-copy invoked by the migration profile's add hook
```

These files are **examples**: copy the relevant pieces into your own repo
root (not into a subdirectory) and adjust service names, images, paths,
volume names, and ports to match your setup. Note that `zgt.config.yaml`
must live at the **main worktree's repo root** so all linked worktrees
inherit it.

## Topology

- The shared DB lives in the **main worktree's** compose project and is the
  only thing on it that needs to be persistent.
- Feature worktrees run only `api` and `envoy`, in their own compose project
  (`COMPOSE_PROJECT_NAME = <repo>-<branch>`). They talk to the shared DB via
  whatever network plumbing your environment supports (external compose
  network, `host.docker.internal`, etc.) - this lives outside `zgt`.
- A `migration` profile (selected with `zgt add --profile migration`)
  spawns an **isolated DB** in the same compose project as the worktree's
  api/envoy, with its Docker volume cloned from the main DB's volume so
  destructive migration tests cannot corrupt the shared DB.

## How it works

### Compose service profiles (docker's own feature)

`docker-compose.yaml` uses per-service `profiles: [db]` on the `db` service.
This means `docker compose up -d api envoy` does **not** start the `db`
service, which avoids eagerly creating an empty
`<COMPOSE_PROJECT_NAME>_pgdata` volume that db-clone.sh would later refuse to
overwrite. Run the DB explicitly with `docker compose --profile db up -d db`.

### Top-level zgt config (default behaviour)

- `COMPOSE_PROJECT_NAME = "<repo>-<branch>"` from `zgt env`.
- `DB_PROJECT = "<repo>-main"` records which compose project owns the shared DB.
- `DB_HOST = "db"` - api reaches the DB by its docker service name; you are
  responsible for wiring cross-project docker networking if api lives in a
  different compose project from the shared DB. Outside the scope of this
  example.
- `hooks.add` runs `docker compose up -d api envoy` (no DB - it is gated by
  compose service profile).
- `hooks.rm` runs `docker compose down`.

### `migration` profile

Selected with `zgt add <branch> --profile migration`. Persists in zgt state
so subsequent `zgt env`, `zgt rm`, and `hooks.*` invocations keep using the
profile for that worktree.

- Sets `DB_HOST = "db"` and `DB_SOURCE_PROJECT = "<repo>-main"` so api talks
  to the isolated DB inside the same compose project and `scripts/db-clone.sh`
  knows which main DB volume to clone.
- Sets `DB_ISOLATED=1` as a hint flag consumed by your own scripts.
- Appends an `add` hook that runs `db-clone.sh && docker compose --profile db
  up -d db api envoy`. Because profile hooks are appended AFTER the
  top-level hooks, api/envoy are already up by this point (the re-invocation
  is a no-op).
- Appends an `rm` hook that runs `docker compose --profile db down -v` to
  tear down **and** wipe the isolated DB volume so migration detritus does
  not leak onto disk between worktrees.

### scripts/db-clone.sh

Copies the main DB's Docker volume (`<DB_SOURCE_PROJECT>_pgdata`) into the
current worktree's volume (`<COMPOSE_PROJECT_NAME>_pgdata`) using `cp -a`
inside an alpine container, then strips `postmaster.pid` so the new
postgres container starts cleanly via crash recovery.

Refuses to overwrite an existing destination volume and refuses to run if
the source volume is missing.

## Usage from the main worktree

```sh
# On the main worktree:
docker compose --profile db up -d db     # always-on shared DB

# Create a normal feature worktree (uses shared DB):
zgt add feat/foo

# Create a migration worktree with its own DB clone:
zgt add feat/migrate-bar --profile migration

# In any worktree:
eval "$(zgt env)"                       # exports COMPOSE_PROJECT_NAME, DB_* etc.
docker compose ps                       # see this worktree's services
```

## Caveats

- Make sure `zgt.config.yaml` is in your `.gitignore` so it does not get
  checked in (similar to `.env`). `zgt init` does this for you.
- Mixed-case profile names are supported but stored lowercased internally
  (Viper lowercases YAML map keys). Lookups via `--profile` flags and
  `{{.Profile}}` placeholders are case-insensitive.
- The db-clone.sh approach uses `cp -a` on a live PGDATA, which relies on
  Postgres' crash recovery on next start. This is fine for most migration
  tests; if you need a guaranteed-consistent clone, stop the source DB
  before copying, or substitute `pg_dump`/`pg_restore` for the script.