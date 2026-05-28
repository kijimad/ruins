//go:build !steam

package steam

// steamタグがないときは何もしない
func Init() error {
	return nil
}
