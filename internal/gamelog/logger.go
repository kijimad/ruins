package gamelog

import (
	"fmt"
	"image/color"

	"github.com/kijimaD/ruins/internal/consts"
)

// Logger はメソッドチェーンで色付きログを作成
type Logger struct {
	currentColor color.RGBA
	fragments    []LogFragment
	store        *SafeSlice
}

// New は指定されたストアでLoggerを作成
// 本番: New(FieldLog) など、グローバルストアを渡す
// テスト: New(testStore) など、ローカルストアを渡す
func New(store *SafeSlice) *Logger {
	return &Logger{
		currentColor: consts.ColorWhite,
		fragments:    make([]LogFragment, 0),
		store:        store,
	}
}

// ColorRGBA は直接color.RGBAを設定
func (l *Logger) ColorRGBA(c color.RGBA) *Logger {
	l.currentColor = c
	return l
}

// Append は現在の色でテキストを追加
func (l *Logger) Append(text interface{}) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: l.currentColor,
		Text:  textStr,
	})
	return l
}

// Log はログを出力（ストアは初期化時に指定済み）
func (l *Logger) Log() {
	l.appendToLog(l.store)
}

// NPCName はNPC名を黄色で追加
func (l *Logger) NPCName(name interface{}) *Logger {
	nameStr := fmt.Sprintf("%v", name)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorYellow,
		Text:  nameStr,
	})
	return l
}

// ItemName はアイテム名をシアン色で追加
func (l *Logger) ItemName(item interface{}) *Logger {
	itemStr := fmt.Sprintf("%v", item)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorCyan,
		Text:  itemStr,
	})
	return l
}

// Damage はダメージ数値を赤色で追加
func (l *Logger) Damage(damage int) *Logger {
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorRed,
		Text:  fmt.Sprintf("%d", damage),
	})
	return l
}

// PlayerName はプレイヤー名を緑色で追加
func (l *Logger) PlayerName(name interface{}) *Logger {
	nameStr := fmt.Sprintf("%v", name)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorGreen,
		Text:  nameStr,
	})
	return l
}

// === ゲーム固有プリセット関数群 ===

// Success は成功メッセージを緑色で追加
func (l *Logger) Success(text interface{}) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorGreen,
		Text:  textStr,
	})
	return l
}

// Warning は警告メッセージを黄色で追加
func (l *Logger) Warning(text interface{}) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorYellow,
		Text:  textStr,
	})
	return l
}

// Error はエラーメッセージを赤色で追加
func (l *Logger) Error(text interface{}) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorRed,
		Text:  textStr,
	})
	return l
}

// Location は場所名をオレンジ色で追加
func (l *Logger) Location(location interface{}) *Logger {
	locationStr := fmt.Sprintf("%v", location)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorOrange,
		Text:  locationStr,
	})
	return l
}

// Action はアクション名を紫色で追加
func (l *Logger) Action(action interface{}) *Logger {
	actionStr := fmt.Sprintf("%v", action)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorPurple,
		Text:  actionStr,
	})
	return l
}

// Money は金額を黄色で追加
// 呼び出し側でFormatCurrencyを使って文字列化してから渡すこと
func (l *Logger) Money(amount interface{}) *Logger {
	amountStr := fmt.Sprintf("%v", amount)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorYellow,
		Text:  amountStr,
	})
	return l
}

// Encounter は敵との遭遇を赤色で追加
func (l *Logger) Encounter(text interface{}) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorRed,
		Text:  textStr,
	})
	return l
}

// Victory は勝利メッセージを緑色で追加
func (l *Logger) Victory(text interface{}) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorGreen,
		Text:  textStr,
	})
	return l
}

// Defeat は敗北メッセージを赤色で追加
func (l *Logger) Defeat(text interface{}) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorRed,
		Text:  textStr,
	})
	return l
}

// Magic は魔法関連を紫色で追加
func (l *Logger) Magic(text interface{}) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorMagenta,
		Text:  textStr,
	})
	return l
}

// System はシステムメッセージを水色で追加
func (l *Logger) System(text interface{}) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: consts.ColorCyan,
		Text:  textStr,
	})
	return l
}

// Build は無名関数を受け取り、メソッドチェーン内でloggerを操作できる
// 複雑なログ構築ロジックをメソッドチェーンに組み込む際に使用
//
// 使用例:
//
//	logger.PlayerName("プレイヤー").
//	       Append(" が ").
//	       Build(func(l *Logger) {
//	           // 複雑な条件分岐やループを含むログ構築
//	           if target.IsPlayer() {
//	               l.PlayerName(targetName)
//	           } else {
//	               l.NPCName(targetName)
//	           }
//	       }).
//	       Append(" を攻撃した。").
//	       Log()
func (l *Logger) Build(builder func(*Logger)) *Logger {
	builder(l)
	return l
}

// 内部ヘルパーメソッド
func (l *Logger) appendToLog(log *SafeSlice) {
	// 複数のフラグメントをログに追加
	if len(l.fragments) == 0 {
		return
	}

	// フラグメントのコピーを作成してLogEntryに追加
	fragmentsCopy := make([]LogFragment, len(l.fragments))
	copy(fragmentsCopy, l.fragments)
	log.pushColoredEntry(LogEntry{Fragments: fragmentsCopy})

	// ログ出力後にフラグメントをクリア
	l.fragments = l.fragments[:0]
}
