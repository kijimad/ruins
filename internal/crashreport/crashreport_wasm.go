//go:build js

package crashreport

// writeRecord は WASM では何もしない。ブラウザにファイルシステムが無く、体験版はセーブも無効なため。
func writeRecord(_ CrashRecord) string { return "" }
