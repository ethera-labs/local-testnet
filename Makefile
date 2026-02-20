EXEC_DIRECTORY=./cmd/localnet
BINARY_NAME=localnet
BINARY_PATH=${EXEC_DIRECTORY}/bin/${BINARY_NAME}
BINARY_DIR=${EXEC_DIRECTORY}/bin
ENCLAVE_NAME=localnet

.PHONY: default
default: help

.PHONY: help
help:
	@echo 'Usage:'
	@echo '  make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_\-]+:.*## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

### Go ###
.PHONY: build
build: ## Build the localnet binary
	go build -o ${BINARY_PATH} ${EXEC_DIRECTORY}
	@if [ -f configs/config.yaml ]; then \
		cp configs/config.yaml ${BINARY_DIR}/config.yaml; \
		echo "Copied config.yaml to ${BINARY_DIR}"; \
	else \
		echo "Warning: configs/config.yaml not found, skipping copy"; \
	fi

.PHONY: run
run: build ## Build and run the localnet binary
	${BINARY_PATH}

.PHONY: clean
clean: clean-observability clean-l2 clean-l1 ## Clean all resources (L1, L2, observability)

.PHONY: stop
stop: stop-observability stop-l2 stop-l1 ## Stop all services (L1, L2, observability)

.PHONY: test
test: ## Run all Go tests
	go test ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run -v ./...

.PHONY: lint-fix
lint-fix: ## Run golangci-lint with auto-fix
	golangci-lint run --fix ./...
######

### L1 ###
.PHONY: run-l1
run-l1: build ## Run the L1 localnet (Kurtosis enclave)
	${BINARY_PATH} l1

.PHONY: show-l1
show-l1: ## Inspect the L1 Kurtosis enclave
	kurtosis enclave inspect ${ENCLAVE_NAME}

.PHONY: stop-l1
stop-l1: ## Stop the L1 Kurtosis enclave
	kurtosis enclave stop ${ENCLAVE_NAME} || true

.PHONY: clean-l1
clean-l1: ## Clean up all Kurtosis enclaves
	kurtosis clean -a

SSV_NODE_COUNT?=4
.PHONY: restart-ssv-nodes
restart-ssv-nodes: ## Restart SSV node services (default: 4, override with SSV_NODE_COUNT=N)
	@echo "Updating SSV Node services. Count: $(SSV_NODE_COUNT) ..."
	@for i in $(shell seq 0 $(shell expr $(SSV_NODE_COUNT) - 1)); do \
		echo "Updating service: ssv-node-$$i"; \
		kurtosis service update $(ENCLAVE_NAME) ssv-node-$$i; \
	done
######

### L2 ###
L2_LABEL=stack=localnet-l2
L2_ARGS?=

.PHONY: run-l2
run-l2: build ## Run the L2 localnet (usage: make run-l2 L2_ARGS="--flashblocks-enabled")
	${BINARY_PATH} l2 $(L2_ARGS)

.PHONY: show-l2
show-l2: ## Show L2 Docker containers
	docker ps -a --filter "label=${L2_LABEL}"

.PHONY: check-l2-altda
check-l2-altda: ## Check Alt-DA server and verify both rollups are running with Alt-DA enabled
	@echo "Checking op-alt-da health endpoint..."
	@curl -fsS http://localhost:3100/health | grep -q '^OK$$'
	@echo "Checking rollup configs include alt_da..."
	@jq -e '.alt_da != null and .alt_da.da_commitment_type != null' .localnet/networks/rollup-a/rollup.json >/dev/null
	@jq -e '.alt_da != null and .alt_da.da_commitment_type != null' .localnet/networks/rollup-b/rollup.json >/dev/null
	@echo "Checking op-node Alt-DA env for both rollups..."
	@for svc in op-node-a op-node-b; do \
		docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' $$svc | grep -q '^OP_NODE_ALTDA_ENABLED=true$$' || { echo "$$svc missing OP_NODE_ALTDA_ENABLED=true"; exit 1; }; \
	done
	@echo "Checking op-batcher Alt-DA env for both rollups..."
	@for svc in op-batcher-a op-batcher-b; do \
		docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' $$svc | grep -q '^OP_BATCHER_ALTDA_ENABLED=true$$' || { echo "$$svc missing OP_BATCHER_ALTDA_ENABLED=true"; exit 1; }; \
	done
	@echo "Alt-DA checks passed for both rollups."

.PHONY: stop-l2
stop-l2: ## Stop L2 Docker containers
	docker compose -f .localnet/docker-compose.yml down || true
	-if [ -f .localnet/docker-compose.flashblocks.yml ]; then docker compose -f .localnet/docker-compose.flashblocks.yml down || true; fi
	-if [ -f .localnet/docker-compose.sidecar.yml ]; then docker compose -f .localnet/docker-compose.sidecar.yml down || true; fi
	-docker rm -f ethera-console 2>/dev/null || true

.PHONY: clean-l2
clean-l2: ## Clean L2 Docker containers and volumes
	@if [ -f .localnet/docker-compose.yml ]; then \
		docker compose -f .localnet/docker-compose.yml down -v || true; \
	fi
	-if [ -f .localnet/docker-compose.flashblocks.yml ]; then docker compose -f .localnet/docker-compose.flashblocks.yml down -v 2>/dev/null || true; fi
	-if [ -f .localnet/docker-compose.sidecar.yml ]; then docker compose -f .localnet/docker-compose.sidecar.yml down -v 2>/dev/null || true; fi
	-if [ -f .localnet/docker-compose.frontend.yml ]; then docker compose -f .localnet/docker-compose.frontend.yml down -v 2>/dev/null || true; fi
	-docker ps -aq --filter "label=${L2_LABEL}" | xargs -r docker rm -f
	-docker rm -f publisher op-geth-a op-geth-b op-node-a op-node-b op-batcher-a op-batcher-b op-rbuilder-a op-rbuilder-b rollup-boost-a rollup-boost-b sidecar-a sidecar-b ethera-console 2>/dev/null || true
	docker volume ls -q | grep -E "(rollup-a|rollup-b|blockscout|op-rbuilder)" | xargs -r docker volume rm
	rm -rf ./.localnet/state ./.localnet/networks ./.localnet/compiled-contracts ./.localnet/docker-compose.yml ./.localnet/docker-compose.blockscout.yml ./.localnet/docker-compose.flashblocks.yml ./.localnet/docker-compose.sidecar.yml ./.localnet/docker-compose.frontend.yml ./.localnet/.tmp ./.localnet/registry ./.cache

.PHONY: clean-l2-full
clean-l2-full: clean-l2 ## Full L2 cleanup including Docker images
	rm -rf ./.localnet/services
	docker images -q "local/publisher" | xargs -r docker rmi -f
	docker images -q "local/op-geth" | xargs -r docker rmi -f
	docker images -q "local/op-rbuilder" | xargs -r docker rmi -f
	docker images -q "local/sidecar" | xargs -r docker rmi -f
	docker images -q "local/ethera-console" | xargs -r docker rmi -f
	docker images -q "us-docker.pkg.dev/oplabs-tools-artifacts/images/op-node" | xargs -r docker rmi -f
	docker images -q "us-docker.pkg.dev/oplabs-tools-artifacts/images/op-batcher" | xargs -r docker rmi -f
	docker images -q "us-docker.pkg.dev/oplabs-tools-artifacts/images/op-proposer" | xargs -r docker rmi -f
	docker images -q "us-docker.pkg.dev/oplabs-tools-artifacts/images/op-deployer" | xargs -r docker rmi -f

.PHONY: run-l2-compile
run-l2-compile: build ## Compile L2 contracts
	${BINARY_PATH} l2 compile

.PHONY: run-frontend
run-frontend: ## Start Ethera Labs Console (cd frontend && bun run dev)
	@cd frontend && bun run dev

.PHONY: frontend-install
frontend-install: ## Install frontend dependencies
	@cd frontend && bun install

SERVICE?=all
.PHONY: run-l2-deploy
run-l2-deploy: build ## Deploy L2 services (usage: make run-l2-deploy SERVICE=op-geth)
	${BINARY_PATH} l2 deploy $(SERVICE)

######

### Observability ###
OBSERVABILITY_LABEL=stack=localnet-observability

.PHONY: run-observability
run-observability: build ## Run the observability stack (Grafana, Prometheus, Loki, Tempo, Alloy)
	${BINARY_PATH} observability

.PHONY: show-observability
show-observability: ## Show observability Docker containers
	docker ps -a --filter "label=${OBSERVABILITY_LABEL}"

.PHONY: stop-observability
stop-observability: ## Stop observability Docker containers
	docker ps -aq --filter "label=${OBSERVABILITY_LABEL}" | xargs -r docker stop

.PHONY: clean-observability
clean-observability: ## Clean observability Docker containers
	docker ps -aq --filter "label=${OBSERVABILITY_LABEL}" | xargs -r docker rm -f
######

### Docker ###
DOCKER_IMAGE_NAME?=ethera-labs/local-testnet
DOCKER_IMAGE_TAG?=latest

.PHONY: docker-build
docker-build: ## Build Docker image for localnet
	docker build -f build/Dockerfile -t ${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} .

.PHONY: docker-run-l2
docker-run-l2: ## Run L2 in Docker (usage: make docker-run-l2 ARGS="...")
	docker run --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(PWD):/workspace \
		-w /workspace \
		-e HOST_PROJECT_PATH=$(PWD) \
		${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG} l2 $(ARGS)
######
