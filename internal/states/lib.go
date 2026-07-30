package states

import (
	"image"

	w "github.com/kijimaD/ruins/internal/world"
)

// getCenterWinRect はゲームワールドから画面サイズを取得してウィンドウ位置を計算する
// TODO: package移動する
func getCenterWinRect(world w.World) image.Rectangle {
	windowWidth, windowHeight := 400, 400 // ウィンドウサイズの設定

	// worldから実際の画面サイズを取得
	screenWidth := world.Resources.ScreenDimensions.Width
	screenHeight := world.Resources.ScreenDimensions.Height

	// ウィンドウの中心が画面の中心に来るように左上角の座標を計算
	x := screenWidth/2 - windowWidth/2
	y := screenHeight/2 - windowHeight/2

	rect := image.Rect(x, y, x+windowWidth, y+windowHeight)
	return rect
}

// ================

// 共通の文字列定数
const (
	// UI表示用の定数
	TextNoDescription = "説明なし" // アイテムの説明がない場合の表示文字列
	TextClose         = "閉じる"  // メニューやウィンドウを閉じる際の表示文字列
	// メニューアクションのラベル。選択肢の生成と分岐で同じ定数を使い、
	// ラベル変更で switch 分岐が黙って死ぬのを防ぐ
	TextCraft = "合成する" // 合成メニューの合成アクション
	TextBuy   = "購入する" // 商店メニューの購入アクション
	TextSell  = "売却する" // 商店メニューの売却アクション
	TextHire  = "雇用する" // 酒場メニューの雇用アクション
	TextEquip = "装備する" // 装備メニューの装備アクション
)
