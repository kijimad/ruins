package steam

// steamLangToCode は Steam の言語名を内部の言語コードへ写す。
// GetCurrentGameLanguage が返す "english"/"japanese" のような独自名を正規化する。
// 対応言語を増やすときは i18n.SupportedLangs と揃える。
var steamLangToCode = map[string]string{
	"english":  "en",
	"japanese": "ja",
}

// normalizeSteamLang は Steam の言語名を内部の言語コードへ変換する。対応表に無ければ空を返す。
func normalizeSteamLang(steamLang string) string {
	return steamLangToCode[steamLang]
}
