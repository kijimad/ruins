.DEFAULT_GOAL := help

.PHONY: run
run: ## 実行する。スクショのキーを指定している
	RUINS_PROFILE=development \
	go run . play

.PHONY: test
test: ## テストを実行する
	RUINS_LOG_LEVEL=ignore \
	go test -v -cover -shuffle=on ./...

.PHONY: build
build: ## ビルドする
	./scripts/build.sh

.PHONY: vrt
vrt: ## 各ステートでスクショを取得する
	./scripts/vrt.sh

.PHONY: fmt
fmt: ## フォーマットする
	goimports -w .
	npx @taplo/cli format

.PHONY: lint
lint: ## Linterを実行する
	@go build -o /dev/null . # buildが通らない状態でlinter実行するとミスリードなエラーが出るので先に試す
	@golangci-lint run -v ./...
	@if deadcode -test ./... 2>&1 | grep -q "unreachable func"; then \
		exit 1; \
	fi

.PHONY: gendata
gendata: ## 現在の設定でデータファイルを生成する
	go run . generate-item-doc
	go run . generate-enemy-doc
	UPDATE_GOLDEN=1 RUINS_LOG_LEVEL=ignore \
	go test ./... -run Golden -v

.PHONY: aseprite
aseprite: ## asepriteでパッキングする。画像の変更を反映したら実行する
	@./scripts/pack.sh

.PHONY: toolsinstall
toolsinstall: ## 開発ツールをインストールする
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install golang.org/x/tools/cmd/deadcode@latest
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.2.2)
	@npm install
	@./scripts/setup-hooks.sh

.PHONY: check
check: fmt build test lint ## 一気にチェックする

# ================

.PHONY: memp
memp: ## 実行毎に保存しているプロファイルを見る
	go tool pprof mem.pprof

.PHONY: pprof
pprof: ## サーバ経由で取得したプロファイルを見る。起動中でなければならない
	go build .
	go tool pprof ruins "http://localhost:6060/debug/pprof/profile?seconds=10"

# ================

.PHONY: help
help: ## ヘルプを表示する
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
