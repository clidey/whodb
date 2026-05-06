SERVICE_NAME = whodb
DOCKER_USERNAME ?=
IMAGE_TAG ?= latest
TARGETARCH ?= amd64
PLATFORM ?= docker
GOPROXY ?= https://proxy.golang.org,direct
BAML_RELEASE_BASE_URL ?= https://github.com/boundaryml/baml/releases/download
IMG ?= $(if $(DOCKER_USERNAME),$(DOCKER_USERNAME)/,)$(SERVICE_NAME):$(IMAGE_TAG)

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

.PHONY: build
build: ## Build DataFlow frontend.
	cd dataflow && pnpm run build

.PHONY: run
run: ## Run DataFlow dev server from host.
	cd dataflow && pnpm run dev

##@ Docker

.PHONY: docker-build
docker-build: ## Build docker image.
	docker buildx build -f core/Dockerfile --platform linux/$(TARGETARCH) --build-arg VERSION=$(IMAGE_TAG) --build-arg TARGETARCH=$(TARGETARCH) --build-arg PLATFORM=$(PLATFORM) --build-arg GOPROXY=$(GOPROXY) --build-arg BAML_RELEASE_BASE_URL=$(BAML_RELEASE_BASE_URL) -t $(IMG) .

.PHONY: docker-build-no-cache
docker-build-no-cache: ## Build docker image without cache.
	docker buildx build --no-cache -f core/Dockerfile --platform linux/$(TARGETARCH) --build-arg VERSION=$(IMAGE_TAG) --build-arg TARGETARCH=$(TARGETARCH) --build-arg PLATFORM=$(PLATFORM) --build-arg GOPROXY=$(GOPROXY) --build-arg BAML_RELEASE_BASE_URL=$(BAML_RELEASE_BASE_URL) -t $(IMG) .

.PHONY: docker-push
docker-push: ## Push docker image.
	docker push $(IMG)

.PHONY: docker-build-push
docker-build-push: ## Build and push docker image.
	docker buildx build -f core/Dockerfile --platform linux/$(TARGETARCH) --build-arg VERSION=$(IMAGE_TAG) --build-arg TARGETARCH=$(TARGETARCH) --build-arg PLATFORM=$(PLATFORM) --build-arg GOPROXY=$(GOPROXY) --build-arg BAML_RELEASE_BASE_URL=$(BAML_RELEASE_BASE_URL) -t $(IMG) --push .

.PHONY: docker-build-push-no-cache
docker-build-push-no-cache: ## Build docker image without cache and push it.
	docker buildx build --no-cache -f core/Dockerfile --platform linux/$(TARGETARCH) --build-arg VERSION=$(IMAGE_TAG) --build-arg TARGETARCH=$(TARGETARCH) --build-arg PLATFORM=$(PLATFORM) --build-arg GOPROXY=$(GOPROXY) --build-arg BAML_RELEASE_BASE_URL=$(BAML_RELEASE_BASE_URL) -t $(IMG) --push .
