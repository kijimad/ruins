package i18n

import (
	"embed"
	"fmt"

	"github.com/leonelquinteros/gotext"
)

//go:embed locale/*.po
var localeFS embed.FS

// defaultLang は源泉言語。原文が英語なので en にする。実行時の表示言語は config が選ぶ。
const defaultLang = "en"

// Translator は現在言語の訳を引く。gotext の Po を包む薄い層で、グローバル state を持たない。
//
// 単一ゴルーチンからの利用を前提にする。T と SetLanguage の並行呼び出しは保護しない。
// 各ワールドが独立した Translator を持つので、共有しなければレースは起きない。
type Translator struct {
	lang string
	po   *gotext.Po
}

// New は指定言語の Translator を作る。ja は埋め込みの ja.po を読み、en は PO を持たず原文を返す。
func New(lang string) (*Translator, error) {
	tr := &Translator{}
	if err := tr.SetLanguage(lang); err != nil {
		return nil, err
	}
	return tr, nil
}

// NewDefault は源泉言語 en の Translator を返す。en は原文を返すので解析に依存せず失敗しないが、
// 将来の既定変更に備えて解析失敗は panic する。解析可能性はテストで担保する。
func NewDefault() *Translator {
	tr, err := New(defaultLang)
	if err != nil {
		panic(fmt.Sprintf("i18n: 既定言語の初期化に失敗した: %v", err))
	}
	return tr
}

// SetLanguage は言語を切り替える。対応言語以外はエラーにする。
func (tr *Translator) SetLanguage(lang string) error {
	po := gotext.NewPo()
	switch lang {
	case "en":
		// 原文が英語なので PO は空にする。Get は未訳フォールバックで原文を返す。
	case "ja":
		data, err := localeFS.ReadFile("locale/ja.po")
		if err != nil {
			return fmt.Errorf("ja.po の読み込みに失敗した: %w", err)
		}
		// gotext の Parse はエラーを返さず内部で握り潰すので、解析失敗はここで検知できない。
		// ja.po の妥当性はテストで担保する。
		po.Parse(data)
	default:
		return fmt.Errorf("未対応の言語: %s", lang)
	}
	tr.po = po
	tr.lang = lang
	return nil
}

// T は英語 msgid に対応する現在言語の訳を返す。未訳なら原文を返す。
//
// gotext の Get は可変引数で printf 整形するため、直接呼ぶと go vet が非定数の書式文字列と誤検知する。
// 整形しないのでメソッド値を介して呼び回避する。
func (tr *Translator) T(msgid string) string {
	get := tr.po.Get
	return get(msgid)
}

// Language は現在の言語コードを返す。
func (tr *Translator) Language() string {
	return tr.lang
}
