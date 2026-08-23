//go:build !steam

package steam

// Init はsteamタグがないときは何もしない
func Init() error {
	return nil
}

// GameLanguage はsteamタグがないとき ok=false を返す。初期言語は既定へ落ちる
func GameLanguage() (code string, ok bool) {
	return "", false
}
