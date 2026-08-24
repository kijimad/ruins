# Steam へのビルドアップロード。SteamCMD + SteamPipe を Linux CLI から回す。
# 設計は単一ビルド・単一 depot。docs/design/260823004406.md を参照。
#
# 必須の変数。Steamworks で発行された値を渡す:
#   APP_ID      アプリID
#   DEPOT_ID    depot ID
#   STEAM_USER  アップロード用 Steam アカウント名
# 例:
#   make -C docs/steam/upload upload APP_ID=1234560 DEPOT_ID=1234561 STEAM_USER=mybuilder
#
# 初回だけ `make login` で対話ログインし Steam Guard を通す。以降トークンが
# SteamCMD の config.vdf にキャッシュされる。CI では config.vdf を秘匿保存して配置する。
#
# 依存: curl, tar。ビルド段は docker を使う root の build-steam に委譲する。

SHELL := /bin/bash
HERE  := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
ROOT  := $(abspath $(HERE)/../../..)

# SteamCMD 実行体。無ければ make steamcmd で取得する
STEAMCMD ?= $(HERE)/steamcmd/steamcmd.sh

APP_ID     ?=
DEPOT_ID   ?=
STEAM_USER ?=
# 名前付きブランチへ自動反映するとき指定。default は指定不可で Web で確定する
BRANCH     ?=
DESC       ?= CLI build $(shell date +%Y-%m-%dT%H:%M:%S)
# make preview が 1 に上書きしてドライランにする
PREVIEW    ?= 0

BUILD   := $(HERE)/.build
CONTENT := $(BUILD)/content
OUTPUT  := $(BUILD)/output
APP_VDF   := $(BUILD)/app_build.vdf
DEPOT_VDF := $(BUILD)/depot_build.vdf

.DEFAULT_GOAL := help

.PHONY: help
help: ## ターゲット一覧を表示する
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
	  | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'

.PHONY: steamcmd
steamcmd: ## SteamCMD を取得する
	@mkdir -p $(HERE)/steamcmd
	@test -f $(HERE)/steamcmd/steamcmd.sh \
	  || curl -sqL https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz \
	     | tar zxf - -C $(HERE)/steamcmd
	@echo "steamcmd: $(STEAMCMD)"

.PHONY: login
login: ## 初回の対話ログイン。Steam Guard を通しトークンをキャッシュする
	@test -n "$(STEAM_USER)" || { echo "STEAM_USER を指定する"; exit 1; }
	$(STEAMCMD) +login $(STEAM_USER) +quit

.PHONY: stage
stage: ## Steam 向けにビルドし content/ へ配置する。root の build-steam に委譲する
	$(MAKE) -C $(ROOT) build-steam
	@rm -rf $(CONTENT) && mkdir -p $(CONTENT)
	@cp $(ROOT)/bin/ruins_linux_amd64_steam       $(CONTENT)/
	@cp $(ROOT)/bin/ruins_windows_amd64_steam.exe $(CONTENT)/
	@echo "staged:"; ls -l $(CONTENT)

.PHONY: render
render: ## VDF テンプレートを実値へ展開する
	@mkdir -p $(BUILD)
	@sed -e 's#@DEPOT_ID@#$(DEPOT_ID)#g' -e 's#@CONTENT@#$(CONTENT)#g' \
	     $(HERE)/depot_build.vdf.in > $(DEPOT_VDF)
	@sed -e 's#@APP_ID@#$(APP_ID)#g' -e 's#@DEPOT_ID@#$(DEPOT_ID)#g' \
	     -e 's#@CONTENT@#$(CONTENT)#g' -e 's#@OUTPUT@#$(OUTPUT)#g' \
	     -e 's#@DESC@#$(DESC)#g' -e 's#@SETLIVE@#$(BRANCH)#g' \
	     -e 's#@PREVIEW@#$(PREVIEW)#g' -e 's#@DEPOT_VDF@#$(DEPOT_VDF)#g' \
	     $(HERE)/app_build.vdf.in > $(APP_VDF)
	@echo "rendered: $(APP_VDF)"

.PHONY: check-vars
check-vars:
	@test -n "$(APP_ID)"     || { echo "APP_ID を指定する";   exit 1; }
	@test -n "$(DEPOT_ID)"   || { echo "DEPOT_ID を指定する"; exit 1; }
	@test -n "$(STEAM_USER)" || { echo "STEAM_USER を指定する"; exit 1; }

.PHONY: preview
preview: PREVIEW := 1
preview: check-vars stage render ## アップロードせず内容だけ検証する。ドライラン
	$(STEAMCMD) +login $(STEAM_USER) +run_app_build $(APP_VDF) +quit

.PHONY: upload
upload: check-vars stage render ## 実際にアップロードする。default 反映は Web で確定する
	$(STEAMCMD) +login $(STEAM_USER) +run_app_build $(APP_VDF) +quit
	@echo "アップロード完了。default へ上げるには partner サイトの Builds ページで確定する"

.PHONY: clean
clean: ## 生成物を消す
	rm -rf $(BUILD)
