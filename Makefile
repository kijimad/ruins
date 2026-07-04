.DEFAULT_GOAL := help

# bwrapが利用可能か実行時にテストする。CI環境ではunprivileged user namespaceが制限されていて使えない場合がある
BWRAP_CMD := $(shell bwrap --dev-bind / / --tmpfs /dev/input -- true 2>/dev/null && echo "bwrap --dev-bind / / --tmpfs /dev/input --")

.PHONY: run
run: ## 実行する。スクショのキーを指定している
	RUINS_PROFILE=development \
	go run . play

.PHONY: editor
editor: ## ゲームデータエディタを起動する
	npm run --prefix editor-ui dev

.PHONY: test
test: ## テストを実行する。COUNT=N で繰り返し実行できる（デフォルト1）
	# editor-ui/node_modules 内にGoパッケージが含まれるため除外する必要がある
	# bwrap: /dev/input を隠してebitenのgamepad初期化エラー(EINTR)を防ぐ
	# xvfb-run: ebitenのゴールデンテストがウィンドウを開くのを防ぐ
	RUINS_LOG_LEVEL=ignore \
	$(BWRAP_CMD) xvfb-run -a go test -v -cover -shuffle=on -timeout=60m -count=$(or $(COUNT),1) \
		$$(go list ./... | grep -v -e /editor-ui/ -e /oapi/)

.PHONY: updategolden
updategolden: ## ゴールデンテスト用の基準画像を生成する
	GOLDIE_UPDATE=1 RUINS_LOG_LEVEL=ignore \
	$(BWRAP_CMD) xvfb-run -a go test ./... -run Golden -v

.PHONY: report
report: ## AIが読みやすい形でカバレッジレポートを表示する
	RUINS_LOG_LEVEL=ignore \
	go test -coverprofile=coverage.out $$(go list ./... | grep -v -e /editor-ui/ -e /oapi/)
	go tool cover -func=coverage.out

.PHONY: build
build: ## ビルドする
	./scripts/build.sh

.PHONY: build-steam
build-steam: ## Steam向けビルドする
	./scripts/build_steam.sh

.PHONY: fmt
fmt: ## フォーマットする
	goimports -w .
	go fix ./...
	npx @taplo/cli format

.PHONY: lint
lint: ## Linterを実行する
	@go build -o /dev/null . # buildが通らない状態でlinter実行するとミスリードなエラーが出るので先に試す
	@golangci-lint run -v ./...
	@if deadcode -test $$(go list ./... | grep -v -e /editor-ui/ -e /oapi/) 2>&1 | grep -q "unreachable func"; then \
		exit 1; \
	fi

.PHONY: aseprite
aseprite: ## asepriteでパッキングする。画像の変更を反映したら実行する
	@./scripts/pack.sh

.PHONY: toolsinstall
toolsinstall: ## 開発ツールをインストールする
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install golang.org/x/tools/cmd/deadcode@latest
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.3
	@sudo apt-get install -y bubblewrap
	@npm install
	@./scripts/setup-hooks.sh

.PHONY: genreadme
genreadme: ## README.tmpl.mdからREADME.mdを生成する
	go run . genreadme

.PHONY: check
check: fmt build test lint ## 一気にチェックする

.PHONY: check-ui
check-ui: ## editor-ui の型チェック・テスト・lintを実行する
	npm run --prefix editor-ui check

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
