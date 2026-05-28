//go:build !steam

package steam

// Init はsteamタグがないときは何もしない
func Init() error {
	return nil
}
