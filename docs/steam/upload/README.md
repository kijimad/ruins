# Steam アップロード

SteamCMD + SteamPipe で Steam へビルドをアップロードする。Linux CLI から完結する。
OS 別に depot を2つに分け、各OSは自分のぶんだけダウンロードする。

## 前提

- Steamworks で発行された **App ID** と **OS 別の depot ID2つ**。Linux 用と Windows 用
- アップロード用の Steam アカウント。専用のビルド用アカウントを推奨する
- `docker` が動くこと。ビルド段は root の `make build-steam` に委譲する
- `curl` と `tar`

## 構成

VDF は静的にコミットしてある。**App ID と depot ID は公開値なので直書きしてよい**。ストアURLや
SteamDB で誰でも見える値で、秘匿すべきは Steam アカウントのパスワードと 2FA だけ。それは
steamcmd の `config.vdf` が持つ。パスは VDF ファイルからの相対で書くので置換は要らない。

- `app_build.vdf`: アプリビルド定義。App ID と2つの depot を参照する
- `depot_linux.vdf` / `depot_windows.vdf`: OS 別の depot 定義。ContentRoot は相対
- `Makefile`: 取得・ログイン・ステージ・アップロードのターゲット
- `content-linux/` `content-windows/`: ステージした実行体の置き場。gitignore 済み
- `.build_output/`: SteamPipe のログ・マニフェスト・キャッシュ。gitignore 済み
- `steamcmd/`: 取得した SteamCMD。gitignore 済み

Linux depot には `bin/ruins_linux_amd64_steam`、Windows depot には `bin/ruins_windows_amd64_steam.exe`
だけを入れる。資産は go:embed でバイナリに焼かれるので depot に別途置かない。
depot ごとの OS 割当は Steamworks の depot 設定で行い、OS 別の起動は launch options で振り分ける。

App ID や depot ID を変えるときは VDF を直接編集する。Windows depot ID は `depot_windows.vdf` と
`app_build.vdf` の TODO を差し替える。

## 使い方

```sh
cd docs/steam/upload

# 1. SteamCMD を取得する。初回だけ
make steamcmd

# 2. 初回の対話ログイン。Steam Guard を通しトークンをキャッシュする
make login STEAM_USER=<steamアカウント名>

# 3. まずドライランで内容を検証する。アップロードしない
make preview STEAM_USER=<アカウント>

# 4. アップロードする
make upload STEAM_USER=<アカウント>
```

アップロード後、`default` ブランチへ反映するのは partner サイトの Builds ページで確定する。
`default` は VDF の `SetLive` では指定できない。名前付きブランチへ自動反映したいときは
`app_build.vdf` に `"SetLive" "beta"` を足す。

## Steam Guard / CI

- 初回 `make login` でトークンが SteamCMD の `config/config.vdf` にキャッシュされる。以降パスワード不要
- CI ではこの `config.vdf` を秘匿保存し、ジョブ内で SteamCMD の `config/` へ配置してから
  `make upload` する。GitHub Actions なら `game-ci/steam-deploy` がこの方式をラップしている
- ログインがブロックされたら SteamCMD 対話中に Steam Guard コードを入力する

## 参照

- SteamPipe / Uploading to Steam: https://partner.steamgames.com/doc/sdk/uploading
- Ebitengine の Steam 向けビルド: https://ebitengine.org/ja/documents/steam.html
