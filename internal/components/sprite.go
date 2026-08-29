package components

import (
	"bytes"
	"image"

	// テクスチャは PNG。image.Decode が読めるよう復号器を登録する
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/kijimaD/ruins/assets"
)

// SubImage は画像の部分矩形を切り出す。
// ebiten.Image.SubImage は image.Image を返すが実体は常に *ebiten.Image であり、
// 各所での未チェック型アサーションを避けるためここに集約する
func SubImage(img *ebiten.Image, r image.Rectangle) *ebiten.Image {
	sub, ok := img.SubImage(r).(*ebiten.Image)
	if !ok {
		panic("ebiten.Image.SubImage did not return *ebiten.Image")
	}
	return sub
}

// Sprite は1つ1つの意味をなす画像の位置を示す情報
// 1ファイルに対して複数のスプライトが定義されている
type Sprite struct {
	X      int
	Y      int
	Width  int
	Height int
}

// Texture は複数のスプライトが格納された画像ファイル
type Texture struct {
	Image *ebiten.Image
	// Source は復号したままの CPU 側の画像。GPU のテクスチャはゲームループが始まるまで
	// 画素を読み出せないため、縮小など画素を触る処理はこちらを源にする
	Source image.Image
}

// UnmarshalText fills structure fields from text data
func (t *Texture) UnmarshalText(text []byte) error {
	bs, err := assets.FS.ReadFile(string(text))
	if err != nil {
		return err
	}
	sourceImage, _, err := image.Decode(bytes.NewReader(bs))
	if err != nil {
		return err
	}
	// unmanaged にして内部テクスチャアトラスへ載せない。nearest サンプリングのテクセル境界の
	// 読み分けはアトラス上の配置座標に依存し、配置は画像の生成順で変わるため、managed のままだと
	// 同じ場面でも画素が揺れる。他の描画ソースの unmanaged 化もこの機構が理由で、説明はここへ集約する
	t.Image = ebiten.NewImageFromImageWithOptions(sourceImage, &ebiten.NewImageFromImageOptions{Unmanaged: true})
	t.Source = sourceImage
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
	// Draw options（実行時の描画オプション。serde対象外）
	Options ebiten.DrawImageOptions `json:"-"`
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
