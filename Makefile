include .env
export

LOCAL_BIN:=$(CURDIR)/bin
PATH:=$(LOCAL_BIN):$(PATH)

# HELP =================================================================================================================
# This will output the help for each task
# thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

compose-up: ### Run docker compose
	docker compose up --build -d postgres && docker compose logs -f
.PHONY: compose-up

compose-up-integration-test: ### Run docker compose with integration test
	docker compose up --build --abort-on-container-exit --exit-code-from integration
.PHONY: compose-up-integration-test

compose-down: ### Down docker compose
	docker compose down --remove-orphans
.PHONY: compose-down

run: ### run app
	go mod tidy && go mod download && \
	GIN_MODE=debug CGO_ENABLED=0 go run ./cmd/app
.PHONY: run

docker-rm-volume: ### remove docker volume
	docker volume rm go-clean-template_pg-data
.PHONY: docker-rm-volume

linter-golangci: ### check by golangci linter
	golangci-lint run
.PHONY: linter-golangci

linter-hadolint: ### check by hadolint linter
	git ls-files --exclude='Dockerfile*' --ignored | xargs hadolint
.PHONY: linter-hadolint

linter-dotenv: ### check by dotenv linter
	dotenv-linter
.PHONY: linter-dotenv

test: ### run test
	go test -v -cover -race ./...
.PHONY: test

integration-test: ### run integration-test
	go clean -testcache && go test -v ./integration-test/...
.PHONY: integration-test

mock: ### run mockgen
	mockgen -source ./internal/usecase/ciraconfigs/interfaces.go        -package mocks  -mock_names Repository=MockCIRAConfigsRepository,Feature=MockCIRAConfigsFeature > ./internal/mocks/ciraconfigs_mocks.go
	mockgen -source ./internal/usecase/devices/interfaces.go            -package mocks  -mock_names Repository=MockDeviceManagementRepository,Feature=MockDeviceManagementFeature > ./internal/mocks/devicemanagement_mocks.go
	mockgen -source ./internal/usecase/amtexplorer/interfaces.go        -package mocks  -mock_names Repository=MockAMTExplorerRepository,Feature=MockAMTExplorerFeature,WSMAN=MockAMTExplorerWSMAN > ./internal/mocks/amtexplorer_mocks.go
	mockgen -source ./internal/usecase/devices/wsman/interfaces.go      -package mocks  > ./internal/mocks/wsman_mocks.go
	mockgen -source ./internal/usecase/export/interface.go              -package mocks  > ./internal/mocks/export_mocks.go
	mockgen -source ./internal/usecase/domains/interfaces.go            -package mocks  -mock_names Repository=MockDomainsRepository,Feature=MockDomainsFeature > ./internal/mocks/domains_mocks.go
	mockgen -source ./internal/controller/ws/v1/interface.go            -package mocks  > ./internal/mocks/wsv1_mocks.go
	mockgen -source ./pkg/logger/logger.go                              -package mocks  -mock_names Interface=MockLogger  > ./internal/mocks/logger_mocks.go
	mockgen -source ./internal/usecase/ieee8021xconfigs/interfaces.go   -package mocks  -mock_names Repository=MockIEEE8021xConfigsRepository,Feature=MockIEEE8021xConfigsFeature > ./internal/mocks/ieee8021xconfigs_mocks.go
	mockgen -source ./internal/usecase/profiles/interfaces.go           -package mocks  -mock_names Repository=MockProfilesRepository,Feature=MockProfilesFeature > ./internal/mocks/profiles_mocks.go
	mockgen -source ./internal/usecase/wificonfigs/interfaces.go        -package mocks  -mock_names Repository=MockWiFiConfigsRepository,Feature=MockWiFiConfigsFeature > ./internal/mocks/wificonfigs_mocks.go
	mockgen -source ./internal/usecase/profilewificonfigs/interfaces.go -package mocks  -mock_names Repository=MockProfileWiFiConfigsRepository,Feature=MockProfileWiFiConfigsFeature > ./internal/mocks/profileswificonfigs_mocks.go
	mockgen -source ./internal/app/interface.go                         -package mocks  > ./internal/mocks/app_mocks.go
	
	
.PHONY: mock

migrate-create:  ### create new migration
	migrate create -ext sql -dir /internal/app/migrations 'migrate_name'
.PHONY: migrate-create

migrate-up: ### migration up
	migrate -path /internal/app/migrations -database '$(DB_URL)?sslmode=disable' up
.PHONY: migrate-up

bin-deps:
	GOBIN=$(LOCAL_BIN) go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	GOBIN=$(LOCAL_BIN) go install go.uber.org/mock/mockgen@latest
