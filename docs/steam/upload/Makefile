# Steam へのビルドアップロード。SteamCMD + SteamPipe を Linux CLI から回す。
# 手順・VDF 構成・CI での config.vdf の扱いは README.md を参照する。
#
# 必須の変数:
#   STEAM_USER  アップロード用 Steam アカウント名
# 例:
#   make -C docs/steam/upload upload STEAM_USER=mybuilder
#
# 依存: curl, tar, sed。ビルド段は docker を使う root の build-steam に委譲する。

SHELL := /bin/bash
HERE  := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
ROOT  := $(abspath $(HERE)/../../..)

# SteamCMD 実行体。無ければ make steamcmd で取得する
STEAMCMD   ?= $(HERE)/steamcmd/steamcmd.sh
STEAM_USER ?=

APP_VDF         := $(HERE)/app_build.vdf
PREVIEW_VDF     := $(HERE)/.app_build.preview.vdf
CONTENT_LINUX   := $(HERE)/content-linux
CONTENT_WINDOWS := $(HERE)/content-windows
# app_build.vdf の "BuildOutput" と一致させる。SteamPipe のログ・キャッシュ置き場
BUILD_OUTPUT    := $(HERE)/.build_output

.DEFAULT_GOAL := help

.PHONY: steamcmd
steamcmd: ## SteamCMD を取得する
	@mkdir -p $(HERE)/steamcmd
	@test -f $(HERE)/steamcmd/steamcmd.sh \
	  || curl -sqL https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz \
	     | tar zxf - -C $(HERE)/steamcmd
	@echo "steamcmd: $(STEAMCMD)"

.PHONY: login
login: check-user ## 初回の対話ログイン。Steam Guard を通しトークンをキャッシュする
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

.PHONY: check-user
check-user:
	@test -n "$(STEAM_USER)" || { echo "STEAM_USER を指定する"; exit 1; }

.PHONY: preview
preview: check-user stage ## アップロードせず内容だけ検証する。ドライラン
	@# Preview を 1 にした一時 VDF を作る。相対パスを保つため VDF と同じディレクトリへ置く
	@sed -E 's/"Preview"[[:space:]]+"0"/"Preview"     "1"/' $(APP_VDF) > $(PREVIEW_VDF)
	$(STEAMCMD) +login $(STEAM_USER) +run_app_build $(PREVIEW_VDF) +quit

.PHONY: upload
upload: check-user stage ## 実際にアップロードする。default 反映は Web で確定する
	$(STEAMCMD) +login $(STEAM_USER) +run_app_build $(APP_VDF) +quit
	@echo "アップロード完了。default へ上げるには partner サイトの Builds ページで確定する"

.PHONY: clean
clean: ## 生成物を消す
	rm -rf $(CONTENT_LINUX) $(CONTENT_WINDOWS) $(BUILD_OUTPUT) $(PREVIEW_VDF)

.PHONY: help
help: ## ターゲット一覧を表示する
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
	  | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}'
