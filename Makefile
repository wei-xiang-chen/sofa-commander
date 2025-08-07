# Makefile for Sofa Commander

# 變數定義
IMAGE_NAME ?= sofa-commander
TAG ?= latest
DOCKER_USERNAME ?= your-dockerhub-username

# 預設目標
.DEFAULT_GOAL := help

.PHONY: help
help: ## 顯示幫助資訊
	@echo "Sofa Commander - 可用的命令："
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## 構建 Docker image
	docker build -t $(IMAGE_NAME):$(TAG) .

.PHONY: run
run: ## 運行 Docker container
	docker run -p 80:80 --env-file .env $(IMAGE_NAME):$(TAG)

.PHONY: run-detached
run-detached: ## 在背景運行 Docker container
	docker run -d -p 80:80 --env-file .env --name $(IMAGE_NAME) $(IMAGE_NAME):$(TAG)

.PHONY: stop
stop: ## 停止 Docker container
	docker stop $(IMAGE_NAME) || true
	docker rm $(IMAGE_NAME) || true

.PHONY: logs
logs: ## 查看 Docker container 日誌
	docker logs -f $(IMAGE_NAME)

.PHONY: shell
shell: ## 進入 Docker container shell
	docker exec -it $(IMAGE_NAME) /bin/sh

.PHONY: clean
clean: ## 清理 Docker images 和 containers
	docker stop $(IMAGE_NAME) || true
	docker rm $(IMAGE_NAME) || true
	docker rmi $(IMAGE_NAME):$(TAG) || true

.PHONY: docker-compose-up
docker-compose-up: ## 使用 docker-compose 啟動服務
	docker-compose up -d

.PHONY: docker-compose-down
docker-compose-down: ## 停止 docker-compose 服務
	docker-compose down

.PHONY: docker-compose-logs
docker-compose-logs: ## 查看 docker-compose 日誌
	docker-compose logs -f

.PHONY: push
push: ## 推送到 Docker Hub
	docker tag $(IMAGE_NAME):$(TAG) $(DOCKER_USERNAME)/$(IMAGE_NAME):$(TAG)
	docker push $(DOCKER_USERNAME)/$(IMAGE_NAME):$(TAG)

.PHONY: test
test: ## 測試 Docker image
	docker run --rm -p 80:80 --env-file .env $(IMAGE_NAME):$(TAG) &
	@sleep 10
	@curl -f http://localhost/ping || (echo "Health check failed" && exit 1)
	@docker stop $$(docker ps -q --filter ancestor=$(IMAGE_NAME):$(TAG)) || true
	@echo "Test passed!"

.PHONY: build-multiarch
build-multiarch: ## 構建多架構 Docker image
	docker buildx build --platform linux/amd64,linux/arm64 -t $(DOCKER_USERNAME)/$(IMAGE_NAME):$(TAG) --push .

.PHONY: dev
dev: ## 開發模式：構建並運行
	$(MAKE) build
	$(MAKE) run-detached
	@echo "服務已啟動在 http://localhost"
	@echo "使用 'make logs' 查看日誌"
	@echo "使用 'make stop' 停止服務" 