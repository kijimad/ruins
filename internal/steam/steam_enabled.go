//go:build steam

package steam

import (
	"fmt"
	"os"

	steamworks "github.com/hajimehoshi/go-steamworks"
)

// AppID はSteamのアプリケーションID。リリース時に正式なIDに変更する
const AppID = 4791810

// initialized は Init が成功したかを表す。GameLanguage が Init より先に呼ばれても、
// 未初期化の SDK に触れてクラッシュしないための番人。
var initialized bool

// Init はSteam APIを初期化する。Steamクライアント経由での起動でない場合はプロセスを終了する
func Init() error {
	if steamworks.RestartAppIfNecessary(AppID) {
		os.Exit(0)
	}
	if err := steamworks.Init(); err != nil {
		return fmt.Errorf("Steamworks APIの初期化に失敗した: %w", err)
	}
	initialized = true
	return nil
}

// GameLanguage は Steam でユーザが選んだ言語を対応言語コードで返す。
// Init 前の呼び出し、対応表に無い言語、取得できない場合は ok=false を返す。
func GameLanguage() (code string, ok bool) {
	if !initialized {
		return "", false
	}
	code = normalizeSteamLang(steamworks.SteamApps().GetCurrentGameLanguage())
	return code, code != ""
}
