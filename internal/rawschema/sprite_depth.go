package rawschema

import "slices"

// depthOrder は段を描く順。先にあるほど先に、つまり下に描く
var depthOrder = []SpriteDepth{
	SpriteDepthFloor,   // 床
	SpriteDepthSnow,    // 地面に重ねる雪の層。3D の描画だけが使い、実体は持たない
	SpriteDepthRemains, // 敵や物が消えた跡
	SpriteDepthDecal,   // 床の模様。例: 土のしみ、ワープホール
	SpriteDepthRug,     // 床に置く低いもの。例: 焚き火、開いた扉
	SpriteDepthTaller,  // 高さのあるもの。例: テーブル、棚、閉じた扉、壁と虚空のタイル。高さのあるタイルは 3D で箱として描く
	SpriteDepthItem,    // 拾えるアイテム。テーブルのような高さのある置物と同じマスにあっても、その上に出す
	SpriteDepthPlayer,  // 操作キャラ。最も手前に描く
}

// Rank は depthOrder の中の位置を返す。セーブから来る未知の段と、段を持たない行の空の値は床の順にする
func (d SpriteDepth) Rank() int {
	if i := slices.Index(depthOrder, d); i >= 0 {
		return i
	}
	return 0
}
