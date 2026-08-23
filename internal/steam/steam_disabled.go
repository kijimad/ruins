//go:build !steam

package steam

// Init はsteamタグがないときは何もしない
func Init() error {
	return nil
}

// GameLanguage はsteamタグがないときは空を返す。ホスト判定はせず初期言語は既定へ落ちる
func GameLanguage() string {
	return ""
}
