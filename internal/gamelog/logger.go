package gamelog

import (
	"fmt"
	"image/color"
)

// Logger はメソッドチェーンで色付きログを作成
type Logger struct {
	currentColor color.RGBA
	fragments    []LogFragment
	store        *SafeSlice
}

// New は指定されたストアでLoggerを作成
// 本番: New(GameLog) など、グローバルストアを渡す
// テスト: New(testStore) など、ローカルストアを渡す
func New(store *SafeSlice) *Logger {
	return &Logger{
		currentColor: ColorWhite,
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
func (l *Logger) Append(text any) *Logger {
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
func (l *Logger) NPCName(name any) *Logger {
	nameStr := fmt.Sprintf("%v", name)
	l.fragments = append(l.fragments, LogFragment{
		Color: ColorYellow,
		Text:  nameStr,
	})
	return l
}

// ItemName はアイテム名をシアン色で追加
func (l *Logger) ItemName(item any) *Logger {
	itemStr := fmt.Sprintf("%v", item)
	l.fragments = append(l.fragments, LogFragment{
		Color: ColorCyan,
		Text:  itemStr,
	})
	return l
}

// Damage はダメージ数値を赤色で追加
func (l *Logger) Damage(damage int) *Logger {
	l.fragments = append(l.fragments, LogFragment{
		Color: ColorRed,
		Text:  fmt.Sprintf("%d", damage),
	})
	return l
}

// PlayerName はプレイヤー名を緑色で追加
func (l *Logger) PlayerName(name any) *Logger {
	nameStr := fmt.Sprintf("%v", name)
	l.fragments = append(l.fragments, LogFragment{
		Color: ColorGreen,
		Text:  nameStr,
	})
	return l
}

// === ゲーム固有プリセット関数群 ===

// Success は成功メッセージを緑色で追加
func (l *Logger) Success(text any) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: ColorGreen,
		Text:  textStr,
	})
	return l
}

// Warning は警告メッセージを黄色で追加
func (l *Logger) Warning(text any) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: ColorYellow,
		Text:  textStr,
	})
	return l
}

// Error はエラーメッセージを赤色で追加
func (l *Logger) Error(text any) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: ColorRed,
		Text:  textStr,
	})
	return l
}

// Location は場所名をオレンジ色で追加
func (l *Logger) Location(location any) *Logger {
	locationStr := fmt.Sprintf("%v", location)
	l.fragments = append(l.fragments, LogFragment{
		Color: ColorOrange,
		Text:  locationStr,
	})
	return l
}

// Action はアクション名を紫色で追加
func (l *Logger) Action(action any) *Logger {
	actionStr := fmt.Sprintf("%v", action)
	l.fragments = append(l.fragments, LogFragment{
		Color: ColorPurple,
		Text:  actionStr,
	})
	return l
}

// Money は金額を黄色で追加
// 呼び出し側でFormatCurrencyを使って文字列化してから渡すこと
func (l *Logger) Money(amount any) *Logger {
	amountStr := fmt.Sprintf("%v", amount)
	l.fragments = append(l.fragments, LogFragment{
		Color: ColorYellow,
		Text:  amountStr,
	})
	return l
}

// System はシステムメッセージを水色で追加
func (l *Logger) System(text any) *Logger {
	textStr := fmt.Sprintf("%v", text)
	l.fragments = append(l.fragments, LogFragment{
		Color: ColorCyan,
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
