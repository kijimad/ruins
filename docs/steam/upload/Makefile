# Steam へのビルドアップロード。SteamCMD + SteamPipe を Linux CLI から回す。
# OS 別に depot を2つに分ける。各OSは自分のぶんだけダウンロードする。
#
# 必須の変数。Steamworks で発行された値を渡す:
#   APP_ID         アプリID
#   DEPOT_LINUX    Linux 用 depot ID
#   DEPOT_WINDOWS  Windows 用 depot ID
#   STEAM_USER     アップロード用 Steam アカウント名
# 例:
#   make -C docs/steam/upload upload \
#     APP_ID=4791810 DEPOT_LINUX=4791811 DEPOT_WINDOWS=4791812 STEAM_USER=mybuilder
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

APP_ID        ?=
DEPOT_LINUX   ?=
DEPOT_WINDOWS ?=
STEAM_USER    ?=
# 名前付きブランチへ自動反映するとき指定。default は指定不可で Web で確定する
BRANCH        ?=
DESC          ?= CLI build $(shell date +%Y-%m-%dT%H:%M:%S)
# make preview が 1 に上書きしてドライランにする
PREVIEW       ?= 0

BUILD           := $(HERE)/.build
CONTENT_LINUX   := $(BUILD)/content-linux
CONTENT_WINDOWS := $(BUILD)/content-windows
OUTPUT          := $(BUILD)/output
APP_VDF           := $(BUILD)/app_build.vdf
DEPOT_LINUX_VDF   := $(BUILD)/depot_linux.vdf
DEPOT_WINDOWS_VDF := $(BUILD)/depot_windows.vdf

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
stage: ## Steam 向けにビルドし OS 別 content へ配置する。root の build-steam に委譲する
	$(MAKE) -C $(ROOT) build-steam
	@rm -rf $(CONTENT_LINUX) $(CONTENT_WINDOWS)
	@mkdir -p $(CONTENT_LINUX) $(CONTENT_WINDOWS)
	@cp $(ROOT)/bin/ruins_linux_amd64_steam       $(CONTENT_LINUX)/
	@cp $(ROOT)/bin/ruins_windows_amd64_steam.exe $(CONTENT_WINDOWS)/
	@echo "staged linux:";   ls -l $(CONTENT_LINUX)
	@echo "staged windows:"; ls -l $(CONTENT_WINDOWS)

.PHONY: render
render: ## VDF テンプレートを実値へ展開する
	@mkdir -p $(BUILD)
	@sed -e 's#@DEPOT_ID@#$(DEPOT_LINUX)#g'   -e 's#@CONTENT@#$(CONTENT_LINUX)#g'   $(HERE)/depot_build.vdf.in > $(DEPOT_LINUX_VDF)
	@sed -e 's#@DEPOT_ID@#$(DEPOT_WINDOWS)#g' -e 's#@CONTENT@#$(CONTENT_WINDOWS)#g' $(HERE)/depot_build.vdf.in > $(DEPOT_WINDOWS_VDF)
	@sed -e 's#@APP_ID@#$(APP_ID)#g' \
	     -e 's#@DEPOT_LINUX@#$(DEPOT_LINUX)#g'     -e 's#@DEPOT_LINUX_VDF@#$(DEPOT_LINUX_VDF)#g' \
	     -e 's#@DEPOT_WINDOWS@#$(DEPOT_WINDOWS)#g' -e 's#@DEPOT_WINDOWS_VDF@#$(DEPOT_WINDOWS_VDF)#g' \
	     -e 's#@CONTENT_ROOT@#$(BUILD)#g' -e 's#@OUTPUT@#$(OUTPUT)#g' \
	     -e 's#@DESC@#$(DESC)#g' -e 's#@SETLIVE@#$(BRANCH)#g' -e 's#@PREVIEW@#$(PREVIEW)#g' \
	     $(HERE)/app_build.vdf.in > $(APP_VDF)
	@echo "rendered: $(APP_VDF)"

.PHONY: check-vars
check-vars:
	@test -n "$(APP_ID)"        || { echo "APP_ID を指定する";        exit 1; }
	@test -n "$(DEPOT_LINUX)"   || { echo "DEPOT_LINUX を指定する";   exit 1; }
	@test -n "$(DEPOT_WINDOWS)" || { echo "DEPOT_WINDOWS を指定する"; exit 1; }
	@test -n "$(STEAM_USER)"    || { echo "STEAM_USER を指定する";    exit 1; }

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
