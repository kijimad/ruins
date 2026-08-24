# Steam アップロード

SteamCMD + SteamPipe で Steam へビルドをアップロードする。Linux CLI から完結する。
設計は単一ビルド・単一 depot。詳細は `docs/design/260823004406.md`。

## 前提

- Steamworks で発行された **App ID** と **depot ID**
- アップロード用の Steam アカウント。専用のビルド用アカウントを推奨する
- `docker` が動くこと。ビルド段は root の `make build-steam` に委譲する
- `curl` と `tar`

## 構成

- `app_build.vdf.in` / `depot_build.vdf.in`: SteamPipe 定義のテンプレート。`@...@` を Makefile が実値へ展開する。秘匿値をコミットしないためテンプレートにしている
- `Makefile`: 取得・ログイン・ステージ・アップロードのターゲット
- `.build/`: 展開後の VDF、ステージした実行体、ログの置き場。gitignore 済み
- `steamcmd/`: 取得した SteamCMD。gitignore 済み

depot の中身は `bin/ruins_linux_amd64_steam` と `bin/ruins_windows_amd64_steam.exe` の2実行体だけ。
資産は go:embed でバイナリに焼かれるので depot に別途置かない。OS 別の起動は Steamworks の
launch options で振り分ける。

## 使い方

```sh
cd docs/steam/upload

# 1. SteamCMD を取得する。初回だけ
make steamcmd

# 2. 初回の対話ログイン。Steam Guard を通しトークンをキャッシュする
make login STEAM_USER=<ビルド用アカウント>

# 3. まずドライランで内容を検証する。アップロードしない
make preview APP_ID=<AppID> DEPOT_ID=<DepotID> STEAM_USER=<アカウント>

# 4. アップロードする
make upload APP_ID=<AppID> DEPOT_ID=<DepotID> STEAM_USER=<アカウント>
```

アップロード後、`default` ブランチへ反映するのは partner サイトの Builds ページで確定する。
名前付きブランチへ自動反映するときは `BRANCH=beta` を渡す。`default` は VDF の `SetLive` では
指定できない。

## Steam Guard / CI

- 初回 `make login` でトークンが SteamCMD の `config/config.vdf` にキャッシュされる。以降パスワード不要
- CI ではこの `config.vdf` を秘匿保存し、ジョブ内で SteamCMD の `config/` へ配置してから
  `make upload` する。GitHub Actions なら `game-ci/steam-deploy` がこの方式をラップしている
- ログインがブロックされたら SteamCMD 対話中に Steam Guard コードを入力する

## 参照

- SteamPipe / Uploading to Steam: https://partner.steamgames.com/doc/sdk/uploading
- Ebitengine の Steam 向けビルド: https://ebitengine.org/ja/documents/steam.html
