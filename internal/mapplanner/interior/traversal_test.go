package interior

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// walkFrom は入口から家具を障害物として4近傍 flood し、歩いて届くタイル集合を返す。歩ける床は全部屋の内側床と
// 戸口で、家具の載るタイルは塞がる。装飾や戦利品は歩行を阻まないので通れる。既存の reachableFloor は部屋単位の
// 検査だが、これは戸口を越えて建物全体を1つの連結空間として歩く。
func walkFrom(site Site, placed []Placed) map[Vec]bool {
	blocked := blockingTiles(placed)
	walkable := site.floorSet()
	// 壁は通れない。玄関ポーチの側壁 ExtraWall は部屋の内側床に混じるが in-game では壁として描かれるので、
	// floorSet から差し引く。ここを引かないと壁を通り抜ける歩行到達になり softlock を見逃す
	for _, w := range site.Walls() {
		delete(walkable, w)
	}
	for d := range site.doorSet() {
		walkable[d] = true // 戸口は壁の切れ目で、部屋と部屋を繋ぐ通り道
	}

	reached := map[Vec]bool{}
	queue := []Vec{site.Door}
	reached[site.Door] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, n := range neighbors4(cur) {
			if walkable[n] && !blocked[n] && !reached[n] {
				reached[n] = true
				queue = append(queue, n)
			}
		}
	}
	return reached
}

// roomEnterable は部屋の内側床のうち家具でないタイルが1つでも reached にあるかを返す。1つでもあれば、その部屋へ
// 入って中の戦利品や什器に触れる。
func roomEnterable(room Room, reached map[Vec]bool) bool {
	for _, v := range room.Rect.interiorTiles() {
		if reached[v] {
			return true
		}
	}
	return false
}

// TestFurnishBuilding_全室が入口から家具越しに歩いて到達できる は、完成した多部屋建物が softlock しないことを
// 守る。既存の到達性テストは部屋単位で、部屋内の床が自室の戸口から届くことしか見ない。ここは入口から戸口を
// 越えて全室へ実際に歩けるかを、家具を障害物とみなした建物全体の flood で確かめる。ある部屋が塞がれた戸口や
// 家具で孤立すると、そこへ入れず戦利品も取れない softlock になる。本番サイズの複数施設を多 seed でなめる。
func TestFurnishBuilding_全室が入口から家具越しに歩いて到達できる(t *testing.T) {
	t.Parallel()

	// テンプレ施設(house/store/clinic)だけでなく BSP フォールバック施設(office/depot/lab/骨董/汎用)も
	// なめる。以前は前者しか回しておらず、玄関ポーチが BSP の狭い部屋の戸口を壁で塞ぐ softlock を見逃していた
	for _, fac := range []FacilityKind{facHouse, facStore, facClinic, facOffice, facDepot, facAntique, facLab, ""} {
		for fp := 17; fp <= 20; fp++ { // 本番の建物サイズ
			for seed := range uint64(50) {
				footprint := Rect{X: 0, Y: 0, W: fp, H: fp}
				door := Vec{X: fp / 2, Y: 0}
				site, placed := FurnishBuilding(seed, footprint, door, fac)
				reached := walkFrom(site, placed)
				for _, hr := range site.Rooms {
					require.Truef(t, roomEnterable(hr.Room, reached),
						"%s fp=%d seed=%d: 部屋 %q %+v が入口から歩いて到達できない", fac, fp, seed, hr.Role, hr.Room.Rect)
				}
			}
		}
	}
}

// TestFurnishBuilding_配置は全て footprint 内に収まる は、生成した配置がチャンクの区画からはみ出さないことを
// 守る。外皮の窓や敷地の塀、hero の目玉は建物本体の外側へ座標を計算するので、辺や向きを1つ間違えると footprint
// の外へ prop を置き、隣の区画や街路へ食い込む。全 Placed が footprint 内にあることを多 seed で確かめる。
func TestFurnishBuilding_配置は全てfootprint内に収まる(t *testing.T) {
	t.Parallel()

	for _, fac := range []FacilityKind{facHouse, facStore, facClinic, facOffice, facDepot} {
		for fp := 17; fp <= 20; fp++ {
			for seed := range uint64(30) {
				footprint := Rect{X: 0, Y: 0, W: fp, H: fp}
				door := Vec{X: fp / 2, Y: 0}
				_, placed := FurnishBuilding(seed, footprint, door, fac)
				for _, p := range placed {
					in := p.Pos.X >= footprint.X && p.Pos.X < footprint.X+footprint.W &&
						p.Pos.Y >= footprint.Y && p.Pos.Y < footprint.Y+footprint.H
					assert.Truef(t, in, "%s fp=%d seed=%d: %q が footprint 外 %+v", fac, fp, seed, p.Ref, p.Pos)
				}
			}
		}
	}
}
