//go:build steam

package steam

import (
	"fmt"
	"os"

	steamworks "github.com/hajimehoshi/go-steamworks"
)

// AppID はSteamのアプリケーションID。リリース時に正式なIDに変更する
const AppID = 4791810

// Init はSteam APIを初期化する。Steamクライアント経由での起動でない場合はプロセスを終了する
func Init() error {
	if steamworks.RestartAppIfNecessary(AppID) {
		os.Exit(0)
	}
	if err := steamworks.Init(); err != nil {
		return fmt.Errorf("Steamworks APIの初期化に失敗した: %w", err)
	}
	return nil
}

// GameLanguage は Steam でユーザが選んだ言語を対応言語コードで返す。対応表に無ければ空。
func GameLanguage() string {
	return normalizeSteamLang(steamworks.SteamApps().GetCurrentGameLanguage())
}
