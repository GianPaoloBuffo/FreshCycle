.PHONY: help bootstrap install install-app supabase-start supabase-stop supabase-status supabase-reset app app-android-dev api test test-app test-api

APP_DIR := app
API_DIR := api
ROOT_DIR := $(CURDIR)
NPM_CACHE := $(ROOT_DIR)/.npm-cache
GOPATH_LOCAL := $(ROOT_DIR)/.go
GOCACHE_LOCAL := $(ROOT_DIR)/.go-cache
GOMODCACHE_LOCAL := $(ROOT_DIR)/.go-mod-cache
GRADLE_USER_HOME_LOCAL := $(ROOT_DIR)/.gradle
ANDROID_SDK_ROOT_LOCAL := $(HOME)/Library/Android/sdk
ANDROID_STUDIO_JAVA_HOME := /Applications/Android Studio.app/Contents/jbr/Contents/Home

help:
	@printf "FreshCycle root commands\n\n"
	@printf "  make bootstrap        Copy local env files and install app dependencies\n"
	@printf "  make install          Install monorepo dependencies\n"
	@printf "  make supabase-start   Start the local Supabase stack\n"
	@printf "  make supabase-stop    Stop the local Supabase stack\n"
	@printf "  make supabase-status  Show local Supabase status and credentials\n"
	@printf "  make supabase-reset   Rebuild the local database from migrations and seed\n"
	@printf "  make app             Start the Expo app\n"
	@printf "  make app-android-dev Build and install the local Android dev client\n"
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

app-android-dev:
	cd $(APP_DIR) && JAVA_HOME="$(ANDROID_STUDIO_JAVA_HOME)" PATH="$(ANDROID_STUDIO_JAVA_HOME)/bin:$$PATH" GRADLE_USER_HOME="$(GRADLE_USER_HOME_LOCAL)" ANDROID_HOME="$(ANDROID_SDK_ROOT_LOCAL)" ANDROID_SDK_ROOT="$(ANDROID_SDK_ROOT_LOCAL)" npm run android:dev

api:
	cd $(API_DIR) && GOPATH=$(GOPATH_LOCAL) GOCACHE=$(GOCACHE_LOCAL) GOMODCACHE=$(GOMODCACHE_LOCAL) go run ./cmd/api

test: test-app test-api

test-app:
	cd $(APP_DIR) && npm run typecheck

test-api:
	cd $(API_DIR) && GOPATH=$(GOPATH_LOCAL) GOCACHE=$(GOCACHE_LOCAL) GOMODCACHE=$(GOMODCACHE_LOCAL) go test ./...
