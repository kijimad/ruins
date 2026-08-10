package i18n

import (
	"embed"
	"fmt"
	"slices"

	"github.com/leonelquinteros/gotext"
)

//go:embed locale/*.po
var localeFS embed.FS

// defaultLang は源泉言語。原文が英語なので en にする。未知の言語はここへフォールバックする。
const defaultLang = "en"

// Lang は対応する言語1つのメタデータを表す。
type Lang struct {
	Code  string // 言語コード。"ja" / "en"
	Label string // 表示名の msgid。query.T で現在言語の訳を引く
}

// supportedLangs は対応する言語の一覧。表示順もこの並び。
// 言語検証・UI の選択肢・翻訳カタログの全てがこの1箇所から導出される唯一の定義。
// 言語を足すときはここへ1行足し、源泉言語 en 以外は locale/<code>.po を置く。
var supportedLangs = []Lang{
	{Code: "ja", Label: "Japanese"},
	{Code: "en", Label: "English"},
}

// SupportedLangs は対応する言語の一覧を表示順で返す。呼び出し側の変更が定義へ波及しないよう複製を返す。
func SupportedLangs() []Lang {
	return slices.Clone(supportedLangs)
}

// IsSupportedLang は code が対応する言語コードかを返す。設定値の検証に使う。
func IsSupportedLang(code string) bool {
	return slices.ContainsFunc(supportedLangs, func(l Lang) bool { return l.Code == code })
}

// Catalog は全言語の翻訳を持つ不変マスタ。embed から構築し、以後変わらない。
// Resources が値で保持し、query.T が現在言語とともに引く。RawMaster と同じ、起動時に構築する読み取り専用データ。
// 内部は map なので値コピーしても実体を共有する。読み取り専用なので共有して問題ない。
type Catalog struct {
	langs map[string]*gotext.Po
}

// NewCatalog は対応言語ごとに埋め込み PO を読んでカタログを構築する。言語集合は supportedLangs に従う。
func NewCatalog() Catalog {
	langs := make(map[string]*gotext.Po, len(supportedLangs))
	for _, l := range supportedLangs {
		langs[l.Code] = loadPo(l.Code)
	}
	return Catalog{langs: langs}
}

// loadPo は言語コードの翻訳 PO を返す。源泉言語 en は原文そのものなので空 PO にし、未訳フォールバックで原文を返す。
// それ以外は埋め込みの locale/<code>.po を解析する。
func loadPo(code string) *gotext.Po {
	if code == defaultLang {
		return gotext.NewPo()
	}
	return mustParse("locale/" + code + ".po")
}

// mustParse は埋め込み PO を読んで解析する。埋め込みなので読み込み失敗はビルドの誤りとして panic する。
// gotext の Parse はエラーを返さず内部で握り潰すので、解析失敗はここで検知できない。妥当性はテストで担保する。
func mustParse(path string) *gotext.Po {
	data, err := localeFS.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("i18n: failed to read embedded PO: %s: %v", path, err))
	}
	po := gotext.NewPo()
	po.Parse(data)
	return po
}

// Translate は lang における msgid の訳を返す。未知の言語や未訳は原文の英語へフォールバックする。
// args を渡すと訳を書式として printf 整形する。args が無ければ整形せずそのまま返す。
//
// gotext の Get は書式引数で printf 整形するため、直接呼ぶと go vet が非定数の書式文字列と誤検知する。
// メソッド値を介して呼び回避する。
func (c Catalog) Translate(lang, msgid string, args ...any) string {
	po, ok := c.langs[lang]
	if !ok {
		po = c.langs[defaultLang]
	}
	get := po.Get
	return get(msgid, args...)
}
