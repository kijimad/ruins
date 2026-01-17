package messagedata

// DialogueTable は会話データのテーブル
var DialogueTable = map[string]func(speakerName string) *MessageData{
	"old_soldier_greeting": func(speakerName string) *MessageData {
		// 1ページ目
		msg1 := NewDialogMessage("", speakerName).
			AddText("「あんた、").
			AddKeyword("遺跡").
			AddText("の").
			AddKeyword("潜り").
			AddText(`だろ?

外からこの街に来る異様に若い連中はみんなそうさ。
向こう見ずで破滅的で、...

どうしようもない事情を持ってる。」`)

		// 2ページ目
		msg2 := NewDialogMessage("", speakerName).
			AddText("「あんたは...、そうか、母親が").
			AddKeyword("虚ろ").
			AddText(`か、...。

救えない世の中だな。」`)

		msg1.NextMessages = append(msg1.NextMessages, msg2)
		return msg1
	},
}

// GetDialogue は指定されたキーに対応する会話データを取得する
func GetDialogue(key string, speakerName string) *MessageData {
	if dialogueFunc, ok := DialogueTable[key]; ok {
		return dialogueFunc(speakerName)
	}
	// デフォルトメッセージ
	return NewDialogMessage("...", speakerName)
}
