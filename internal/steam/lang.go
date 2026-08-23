package steam

// steamLangToCode は Steam の言語名を内部の言語コードへ写す。
// GetCurrentGameLanguage は "english"/"japanese" のような Steam 独自の言語名を返すので、
// ここで対応言語コードへ正規化する。対応言語を増やすときは i18n.SupportedLangs と揃える。
// build tag に依らず常にコンパイルし、i18n との整合をテストで固定できるようにする。
var steamLangToCode = map[string]string{
	"english":  "en",
	"japanese": "ja",
}

// normalizeSteamLang は Steam の言語名を内部の言語コードへ変換する。対応表に無ければ空を返す。
func normalizeSteamLang(steamLang string) string {
	return steamLangToCode[steamLang]
}
