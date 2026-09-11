package overworld

import "github.com/kijimaD/ruins/internal/consts"

// マクロ地図に道を重ねるための算出。道タイルを生成せず、road.go と同じ結線をチャンク空間で
// 純関数として再現する。集落中心は WinnerOf で算出でき、経路は「始点の行の水平辺 →
// 終点の列の垂直辺」の L 字なので、各チャンクをどの方角へ抜けるかをビット集合で表せる。

// RoadDir はチャンクを通る道が接続する方角のビット集合。線分描画で、セル中央からどの辺の
// 中点へ線を引くかを表す。0 は道なし。
type RoadDir uint8

const (
	// RoadN は上、Y-1 の隣へ抜ける。
	RoadN RoadDir = 1 << iota
	// RoadS は下、Y+1 の隣へ抜ける。
	RoadS
	// RoadE は右、X+1 の隣へ抜ける。
	RoadE
	// RoadW は左、X-1 の隣へ抜ける。
	RoadW
)

// buildRoadOverlay は窓に重なる範囲の道の接続方角を、絶対チャンク座標をキーに算出する。
// road.go と同じく隣接リージョンの当選集落どうしを L 字で結ぶ。生成を伴わない純関数で、
// 窓外へはみ出すチャンクも含みうるが、読み手が窓内だけを引くので無害。
func buildRoadOverlay(runSeed uint64, win MacroWindow, rows consts.Chunk) map[consts.Coord[consts.Chunk]]RoadDir {
	overlay := map[consts.Coord[consts.Chunk]]RoadDir{}
	rows = max(rows, 1)
	// 道 (pr, pr+1) が占めるチャンク列は当選集落 a.X..b.X で、a.X は pr*Spacing 以上、b.X は
	// (pr+2)*Spacing 未満に収まる。よって窓 [OriginX, OriginX+Cols) に列が掛かりうる道は、窓左端の
	// 属するリージョンの1つ西から窓右端の属するリージョンまで。取りこぼしを防ぐため左右へ1リージョンの
	// 余裕を足す。窓外へ出たチャンクを印しても読み手が引かないので無害
	rLo := floorDiv(win.OriginX, settlementPlacement.Spacing) - 2
	rHi := floorDiv(win.OriginX+win.Cols-1, settlementPlacement.Spacing) + 1
	for pr := rLo; pr <= rHi; pr++ {
		a := settlementPlacement.WinnerOf(runSeed, pr, rows)
		b := settlementPlacement.WinnerOf(runSeed, pr+1, rows)
		markRoadLShape(overlay, a, b)
	}
	return overlay
}

// markRoadLShape は当選集落 a から b への L 字経路のチャンクへ接続方角を積む。分解は road.go の
// drawRoadSegments に合わせ、水平辺は始点 a の行 a.Y、垂直辺は終点 b の列 b.X、角は (b.X, a.Y)。
// 各ステップで進行元と進行先の両チャンクに互いを指すビットを立てる。角は水平ビットと垂直ビットが
// 同一チャンクに積まれて折れになり、複数の道が重なる交差はビットが増えて T・十字になる。
func markRoadLShape(overlay map[consts.Coord[consts.Chunk]]RoadDir, a, b consts.Coord[consts.Chunk]) {
	// 水平辺。始点 a の行に沿って a.X から b.X へ。a.X < b.X なので東進のみだが、符号で一般化する
	stepX := chunkSign(b.X - a.X)
	for x := a.X; x != b.X; x += stepX {
		cur := consts.Coord[consts.Chunk]{X: x, Y: a.Y}
		nxt := consts.Coord[consts.Chunk]{X: x + stepX, Y: a.Y}
		if stepX > 0 {
			overlay[cur] |= RoadE
			overlay[nxt] |= RoadW
		} else {
			overlay[cur] |= RoadW
			overlay[nxt] |= RoadE
		}
	}
	// 垂直辺。終点 b の列に沿って a.Y から b.Y へ。角 (b.X, a.Y) に水平と垂直の両ビットが積まれる
	stepY := chunkSign(b.Y - a.Y)
	for y := a.Y; y != b.Y; y += stepY {
		cur := consts.Coord[consts.Chunk]{X: b.X, Y: y}
		nxt := consts.Coord[consts.Chunk]{X: b.X, Y: y + stepY}
		if stepY > 0 {
			overlay[cur] |= RoadS
			overlay[nxt] |= RoadN
		} else {
			overlay[cur] |= RoadN
			overlay[nxt] |= RoadS
		}
	}
}

// chunkSign は符号を返す。0 のとき 0 を返し、markRoadLShape のループは進まず即座に終わる。
func chunkSign(d consts.Chunk) consts.Chunk {
	switch {
	case d > 0:
		return 1
	case d < 0:
		return -1
	default:
		return 0
	}
}
