.DEFAULT_GOAL := help

.PHONY: toolinstall
toolinstall: ## TypeSpec関連ツールをインストールする
	cd typespec && npm install

.PHONY: compile
compile: ## TypeSpec を OpenAPI にコンパイルする
	cd typespec && npx tsp compile . --config tspconfig.yml

.PHONY: genapi
genapi: ## OpenAPIスキーマからバックエンドコードを生成する
	./scripts/generate-api.sh

.PHONY: genfront
genfront: ## OpenAPIスキーマからフロントエンドコードを生成する
	./scripts/generate-frontend.sh

.PHONY: genall
genall: compile genapi genfront ## すべて生成する

.PHONY: help
help: ## ヘルプを表示する
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
