//go:build !steam

package steam

// Init はsteamタグがないときは何もしない
func Init() error {
	return nil
}

// GameLanguage はsteamタグがないとき ok=false を返す
func GameLanguage() (code string, ok bool) {
	return "", false
}
