.PHONY: help bootstrap install install-app supabase-start supabase-stop supabase-status supabase-reset app api test test-app test-api

APP_DIR := app
API_DIR := api
ROOT_DIR := $(CURDIR)
NPM_CACHE := $(ROOT_DIR)/.npm-cache
GOPATH_LOCAL := $(ROOT_DIR)/.go
GOCACHE_LOCAL := $(ROOT_DIR)/.go-cache
GOMODCACHE_LOCAL := $(ROOT_DIR)/.go-mod-cache

help:
	@printf "FreshCycle root commands\n\n"
	@printf "  make bootstrap        Copy local env files and install app dependencies\n"
	@printf "  make install          Install monorepo dependencies\n"
	@printf "  make supabase-start   Start the local Supabase stack\n"
	@printf "  make supabase-stop    Stop the local Supabase stack\n"
	@printf "  make supabase-status  Show local Supabase status and credentials\n"
	@printf "  make supabase-reset   Rebuild the local database from migrations and seed\n"
	@printf "  make app             Start the Expo app\n"
	@printf "  make api             Start the Go API\n"
	@printf "  make test            Run app and API verification commands\n"

bootstrap:
	@test -f $(APP_DIR)/.env.local || cp $(APP_DIR)/.env.example $(APP_DIR)/.env.local
	@test -f $(API_DIR)/.env.local || cp $(API_DIR)/.env.example $(API_DIR)/.env.local
	@$(MAKE) install
	@printf "Bootstrap complete. Review %s/.env.local and %s/.env.local, then run 'make supabase-start'.\n" "$(APP_DIR)" "$(API_DIR)"

install: install-app

install-app:
	cd $(APP_DIR) && npm_config_cache=$(NPM_CACHE) npm install

supabase-start:
	supabase start

supabase-stop:
	supabase stop

supabase-status:
	supabase status

supabase-reset:
	supabase db reset

app:
	cd $(APP_DIR) && npm run start

api:
	cd $(API_DIR) && GOPATH=$(GOPATH_LOCAL) GOCACHE=$(GOCACHE_LOCAL) GOMODCACHE=$(GOMODCACHE_LOCAL) go run ./cmd/api

test: test-app test-api

test-app:
	cd $(APP_DIR) && npm run typecheck

test-api:
	cd $(API_DIR) && GOPATH=$(GOPATH_LOCAL) GOCACHE=$(GOCACHE_LOCAL) GOMODCACHE=$(GOMODCACHE_LOCAL) go test ./...
