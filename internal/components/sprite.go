package components

import (
	"bytes"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/kijimaD/ruins/assets"
)

// Sprite は1つ1つの意味をなす画像の位置を示す情報
// 1ファイルに対して複数のスプライトが定義されている
type Sprite struct {
	// Horizontal position of the sprite in the sprite sheet
	X int
	// Vertical position of the sprite in the sprite sheet
	Y int
	// Width of the sprite
	Width int
	// Height of the sprite
	Height int
}

// Texture は複数のスプライトが格納された画像ファイル
type Texture struct {
	// Texture image
	Image *ebiten.Image
}

// UnmarshalText fills structure fields from text data
func (t *Texture) UnmarshalText(text []byte) error {
	bs, err := assets.FS.ReadFile(string(text))
	if err != nil {
		return err
	}
	textureImage, _, err := ebitenutil.NewImageFromReader(bytes.NewReader(bs))
	if err != nil {
		return err
	}
	t.Image = textureImage
	return nil
}

// SpriteSheet は画像ファイルであるテクスチャと、その位置ごとの解釈であるスプライトのマッピング
type SpriteSheet struct {
	// スプライトシートのキー名
	Name string
	// 読み込んだ画像データ
	Texture Texture `toml:"texture_image"`
	// 画像に含まれるスプライト辞書
	Sprites map[string]Sprite
}

// SpriteRender component
// コンポーネントはセーブ&ロード時のシリアライズが必要なので、ファイル類をそのまま保存できない
type SpriteRender struct {
	// スプライトシート名(画像データはResourcesから取得)
	SpriteSheetName string
	// スプライトキー名
	SpriteKey string
	// アニメーション用スプライトキー配列。空ならアニメーションなし
	AnimKeys []string
	// 描画順。小さい順に先に(下に)描画する
	Depth DepthNum
	// Draw options
	Options ebiten.DrawImageOptions
}

// DepthNum はオブジェクトの描画順。小さい値を先に描画する
type DepthNum int

const (
	// DepthNumFloor は床。最背面に表示する
	DepthNumFloor DepthNum = iota
	// DepthNumRug は床に置くもの。例: ワープホール、アイテム
	DepthNumRug
	// DepthNumTaller は高さのあるもの。例: 操作対象エンティティ、敵シンボル、壁
	DepthNumTaller
	// DepthNumPlayer は操作キャラを最も手前に表示する
	DepthNumPlayer
)
