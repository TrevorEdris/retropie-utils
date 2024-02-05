BIN=bin
# TODO: Allow for multiple tools in the future
BINARY_NAME=syncer
BINARY_LOCATION=${BIN}/${BINARY_NAME}
MAIN_LOCATION=tools/${BINARY_NAME}/main.go
LOCAL_OUTPUT=_output
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
GIT_HASH=$(shell git rev-parse --short HEAD)
DEV_DOCKER_COMPOSE=docker-compose.dev.yaml

show:
	@echo ${GOOS}
	@echo ${GOARCH}
	@echo ${GIT_HASH}

install-dev-tools:
	@go install github.com/onsi/ginkgo/v2/ginkgo
	@go install github.com/nunnatsa/ginkgolinter/cmd/ginkgolinter@latest

create-bin:
	mkdir -p ./${BIN}

build-linux: create-bin
	GOARCH=amd64 GOOS=linux go build -o ${BINARY_LOCATION}-linux ${MAIN_LOCATION}

build-darwin: create-bin
	GOARCH=amd64 GOOS=darwin go build -o ${BINARY_LOCATION}-darwin ${MAIN_LOCATION}

build-windows: create-bin
	GOARCH=amd64 GOOS=windows go build -o ${BINARY_LOCATION}-windows ${MAIN_LOCATION}

package: build-linux build-darwin build-windows
	zip -r ${BINARY_NAME}-${GIT_HASH}.zip ${BINARY_LOCATION}-windows ${BINARY_LOCATION}-darwin ${BINARY_LOCATION}-linux

build: create-bin
	go build -o ${BINARY_LOCATION} ${MAIN_LOCATION}

clean:
	go clean
	rm -rf ${BIN}
	rm -rf ${BINARY_NAME}*.zip
	rm -rf ${LOCAL_OUTPUT}

prepare_tests:
	mkdir -p ${LOCAL_OUTPUT}

test:
	ginkgo -v ./...

# TODO: Choose between `wslview` and `open`
test_coverage: prepare_tests
	go test -v --cover --covermode=count --coverprofile=${LOCAL_OUTPUT}/testcoverage.cov ./...
	go tool cover -html ${LOCAL_OUTPUT}/testcoverage.cov -o ${LOCAL_OUTPUT}/testcoverage.html
	wslview ${LOCAL_OUTPUT}/testcoverage.html

dep:
	go mod download

vet:
	go vet

# TODO: Address lint issues... oh my so many
lint:
	golangci-lint run --enable-all ./...

.PHONY: dev
dev: ## Run the apps locally and print logs to stdout
	docker-compose -f ${DEV_DOCKER_COMPOSE} up

.PHONY: dev-down
dev-down: ## Stop all containers
	docker-compose -f ${DEV_DOCKER_COMPOSE} down

.PHONY: dev-restart
dev-restart: ## Restart all containers
	docker-compose -f ${DEV_DOCKER_COMPOSE} restart
