package i18n

import (
	"embed"
	"fmt"

	"github.com/leonelquinteros/gotext"
)

//go:embed locale/*.po
var localeFS embed.FS

// defaultLang は源泉言語。原文が英語なので en にする。未知の言語はここへフォールバックする。
const defaultLang = "en"

// Catalog は全言語の翻訳を持つ不変マスタ。embed から構築し、以後変わらない。
// Resources が値で保持し、query.T が現在言語とともに引く。RawMaster と同じ、起動時に構築する読み取り専用データ。
// 内部は map なので値コピーしても実体を共有する。読み取り専用なので共有して問題ない。
type Catalog struct {
	langs map[string]*gotext.Po
}

// NewCatalog は埋め込み PO から全言語のカタログを構築する。en は原文そのものなので空 PO にする。
func NewCatalog() Catalog {
	return Catalog{
		langs: map[string]*gotext.Po{
			// 原文が英語なので PO を持たない。未訳フォールバックで原文をそのまま返す。
			"en": gotext.NewPo(),
			"ja": mustParse("locale/ja.po"),
		},
	}
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
