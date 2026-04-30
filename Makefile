.DEFAULT_GOAL := help

.PHONY: run
run: ## 実行する。スクショのキーを指定している
	RUINS_PROFILE=development \
	go run . play

.PHONY: editor
editor: ## ゲームデータエディタを起動する(コード変更で自動再起動)
	@while true; do \
		setsid go run . editor & PID=$$!; \
		inotifywait -r -e close_write,moved_to --include '\.go$$' internal/editor/ ; \
		kill -- -$$PID 2>/dev/null; wait $$PID 2>/dev/null; \
	done

.PHONY: test
test: ## テストを実行する
	RUINS_LOG_LEVEL=ignore \
	go test -v -cover -shuffle=on ./...

.PHONY: report
report: ## AIが読みやすい形でカバレッジレポートを表示する
	RUINS_LOG_LEVEL=ignore \
	go test -coverprofile=cover.out ./...
	go tool cover -func=cover.out

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
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.3
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
