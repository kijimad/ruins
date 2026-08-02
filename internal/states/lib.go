package states

import (
	"image"

	"github.com/ebitenui/ebitenui/widget"
	w "github.com/kijimaD/ruins/internal/world"
)

// wrapModalRoot は root を画面より一回り小さい中央モーダルとして包む。
// 外周は背景を持たず透明にし、周囲に後ろのフィールドを覗かせる。動詞タブ画面と各メニューで共通に使う。
func wrapModalRoot(root *widget.Container) *widget.Container {
	outer := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(1),
			widget.GridLayoutOpts.Stretch([]bool{true}, []bool{true}),
			widget.GridLayoutOpts.Padding(&widget.Insets{Top: 48, Bottom: 48, Left: 96, Right: 96}),
		)),
	)
	outer.AddChild(root)
	return outer
}

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
