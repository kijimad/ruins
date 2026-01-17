package tabmenu

// Item はメニュー項目を表す
type Item struct {
	ID               string
	Label            string
	AdditionalLabels []string // 追加表示項目（個数、価格など）右側に表示される
	Disabled         bool
	UserData         interface{} // 任意のデータを保持
}
