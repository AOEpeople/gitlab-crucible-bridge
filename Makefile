help:
	@echo "Available targets:"
	@echo "- run: runs the gitlab crucible bridge"

.PHONY: run

SRC = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

gitlab-crucible-bridge: $(SRC)
	go build

run: gitlab-crucible-bridge
	CRUCIBLE_PROJECT_REFRESH_INTERVAL=60 \
	CRUCIBLE_PROJECT_LIMIT=10 \
	./gitlab-crucible-bridge