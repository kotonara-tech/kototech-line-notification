BIN_DIR    := bin
BINARY     := $(BIN_DIR)/notifier
MAIN_PKG   := ./cmd/notifier

GOOS       ?= $(shell go env GOOS)
GOARCH     ?= $(shell go env GOARCH)

.DEFAULT_GOAL := help

.PHONY: help
help: ## コマンド一覧を表示する
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-18s %s\n", $$1, $$2}'

.PHONY: test
test: ## ユニットテストを実行する(外部 API は叩かない)
	go test -race -cover ./...

.PHONY: lint
lint: ## go vet(integration タグ付きのコードも含む)と golangci-lint を実行する
	go vet ./...
	go vet -tags=integration ./...
	golangci-lint run ./...

# 実 API(dev 環境の Sheets / LINE など)を使うテスト。`//go:build integration` を付けたファイルが対象。
# 実行には認証情報(ADC)と SPREADSHEET_ID などの環境変数が必要。dev 環境に対してのみ実行すること。
.PHONY: integration-test
integration-test: ## インテグレーションテストを実行する(dev 環境の実 API を使う)
	go test -race -tags=integration -count=1 ./...

.PHONY: build
build: ## Go バイナリを 1 つビルドする(bin/notifier)。GOOS / GOARCH で上書き可
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags "-s -w" -o $(BINARY) $(MAIN_PKG)

.PHONY: clean
clean: ## ビルド成果物を削除する
	rm -rf $(BIN_DIR)
