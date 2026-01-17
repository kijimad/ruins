package gamelog

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
)

// この例は、Loggerのメソッドチェーンの使い方を示しています
func ExampleLogger_methodChaining() {
	// ローカルログストアを作成
	testLog := NewSafeSlice(FieldLogMaxSize)

	// メソッドチェーンでのログ作成
	New(testLog).
		NPCName("Goblin").
		Append(" attacks you for ").
		Damage(15).
		Append(" damage!").
		Log()

	// 色指定も同様
	New(testLog).
		ColorRGBA(consts.ColorCyan). // Cyan
		Append("John").
		ColorRGBA(consts.ColorWhite).
		Append(" considers attacking ").
		ColorRGBA(consts.ColorCyan).
		Append("Orc").
		Log()

	// アイテムとプレイヤー名
	New(testLog).
		PlayerName("Hero").
		Append(" picks up ").
		ItemName("Iron Sword").
		Append(".").
		Log()

	fmt.Println("Logs created successfully!")
	// Output: Logs created successfully!
}

// カスタム色の例
func ExampleLogger_customColors() {
	// ローカルフィールドログストアを作成
	testFieldLog := NewSafeSlice(FieldLogMaxSize)
	testFieldLog.Clear()

	New(testFieldLog).
		ColorRGBA(consts.ColorPurple).
		Append("Magic spell ").
		ColorRGBA(consts.ColorOrange).
		Append("Fire Bolt").
		ColorRGBA(consts.ColorWhite).
		Append(" hits for ").
		Damage(20).
		Append(" damage!").
		Log()

	// 色付きエントリの取得
	entries := testFieldLog.GetRecentEntries(1)
	fmt.Printf("Entry has %d colored fragments\n", len(entries[0].Fragments))
	// Output: Entry has 5 colored fragments
}

// 連続攻撃のログ例
func ExampleLogger_chainedAttack() {
	// ローカルフィールドログストアを作成
	testFieldLog := NewSafeSlice(FieldLogMaxSize)

	New(testFieldLog).
		NPCName("Orc").
		Append(" attacks ").
		Append("->").
		NPCName("Player").
		Append(" for ").
		Damage(8).
		Log()

	fmt.Println("Chained attack log created!")
	// Output: Chained attack log created!
}
