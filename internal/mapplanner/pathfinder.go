package mapplanner

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
)

// PathFinder はパスファインディング機能を提供する
type PathFinder struct {
	planData *MetaPlan
}

// NewPathFinder はPathFinderを作成する
func NewPathFinder(planData *MetaPlan) *PathFinder {
	return &PathFinder{planData: planData}
}

// fourDirs は上下左右の4方向のタイル差分
var fourDirs = []consts.Coord[consts.Tile]{{X: 0, Y: 1}, {X: 1, Y: 0}, {X: 0, Y: -1}, {X: -1, Y: 0}}

// IsWalkable は指定座標が歩行可能かを判定する
func (pf *PathFinder) IsWalkable(pos consts.Coord[consts.Tile]) bool {
	// 境界チェック
	if pos.X < 0 || pos.X >= pf.planData.Level.TileWidth || pos.Y < 0 || pos.Y >= pf.planData.Level.TileHeight {
		return false
	}

	idx := pf.planData.Level.CoordToIndex(pos)
	return !pf.planData.Tiles[idx].BlockPass
}

// FindPath はBFSでスタート地点からゴールまでのパスを探索する。
// 上下左右の4方向移動のみサポートする。到達できない場合は空を返す
func (pf *PathFinder) FindPath(start, goal consts.Coord[consts.Tile]) []consts.Coord[consts.Tile] {
	if !pf.IsWalkable(start) || !pf.IsWalkable(goal) {
		return nil
	}

	// parent はパス復元用、visited は探索済み。どちらもタイル座標をキーにする
	parent := map[consts.Coord[consts.Tile]]consts.Coord[consts.Tile]{}
	visited := map[consts.Coord[consts.Tile]]bool{start: true}
	queue := []consts.Coord[consts.Tile]{start}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if cur == goal {
			return pf.reconstructPath(parent, start, goal)
		}

		for _, d := range fourDirs {
			next := cur.Add(d)
			// IsWalkable が境界外を false にするので境界チェックは兼ねる
			if visited[next] || !pf.IsWalkable(next) {
				continue
			}
			visited[next] = true
			parent[next] = cur
			queue = append(queue, next)
		}
	}

	return nil
}

// reconstructPath は親マップからスタート→ゴール順のパスを復元する
func (pf *PathFinder) reconstructPath(parent map[consts.Coord[consts.Tile]]consts.Coord[consts.Tile], start, goal consts.Coord[consts.Tile]) []consts.Coord[consts.Tile] {
	var path []consts.Coord[consts.Tile]
	for cur := goal; ; cur = parent[cur] {
		path = append(path, cur)
		if cur == start {
			break
		}
	}

	// スタートからゴールの順序にする
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path
}

// IsReachable はスタート地点からゴール地点まで到達可能かを判定する
func (pf *PathFinder) IsReachable(start, goal consts.Coord[consts.Tile]) bool {
	return len(pf.FindPath(start, goal)) > 0
}

// countReachableFrom は指定位置からBFSで到達可能な歩行可能タイル数を返す
func (pf *PathFinder) countReachableFrom(start consts.Coord[consts.Tile]) int {
	if !pf.IsWalkable(start) {
		return 0
	}

	visited := map[consts.Coord[consts.Tile]]bool{start: true}
	queue := []consts.Coord[consts.Tile]{start}
	count := 0

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		count++

		for _, d := range fourDirs {
			next := cur.Add(d)
			if !visited[next] && pf.IsWalkable(next) {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}

	return count
}

// hasAdjacentFreeTile は隣接4方向のうち少なくとも1つが歩行可能かつ計画済みエンティティがないかを判定する
func (pf *PathFinder) hasAdjacentFreeTile(pos consts.Coord[consts.Tile]) bool {
	for _, d := range fourDirs {
		next := pos.Add(d)
		if pf.IsWalkable(next) && !pf.planData.existPlannedEntityOnTile(next) {
			return true
		}
	}
	return false
}

// minReachableTiles はスポーン位置から到達可能なタイルの最小数。
// 小さな孤立部屋にスポーンしてポータルが至近距離に配置される問題を防ぐ
const minReachableTiles = 100

// FindPlayerStartPosition はプレイヤーの開始位置を探す
func (pf *PathFinder) FindPlayerStartPosition() (consts.Coord[consts.Tile], error) {
	planData := pf.planData

	// SpawnPointsが設定されていればそれを使用（テンプレートマップ用）
	if len(planData.SpawnPoints) > 0 {
		return consts.Coord[consts.Tile]{X: consts.Tile(planData.SpawnPoints[0].X), Y: consts.Tile(planData.SpawnPoints[0].Y)}, nil
	}

	// プロシージャルマップ用: 歩行可能でポータルに到達可能な位置を探す
	width := planData.Level.TileWidth
	height := planData.Level.TileHeight

	// 候補位置を試す（中央から外側へ）
	attempts := []consts.Coord[consts.Tile]{
		{X: width / 2, Y: height / 2},
		{X: width / 4, Y: height / 4},
		{X: 3 * width / 4, Y: height / 4},
		{X: width / 4, Y: 3 * height / 4},
		{X: 3 * width / 4, Y: 3 * height / 4},
	}

	for _, pos := range attempts {
		if pf.isValidSpawnPosition(pos) {
			return pos, nil
		}
	}

	// 見つからない場合は全体をスキャン
	for _i, tile := range planData.Tiles {
		if !tile.BlockPass {
			pos := planData.Level.IndexToCoord(gc.TileIdx(_i))
			if pf.isValidSpawnPosition(pos) {
				return pos, nil
			}
		}
	}

	return consts.Coord[consts.Tile]{}, fmt.Errorf("ポータルに到達可能な歩行可能タイルが見つかりません")
}

// isValidSpawnPosition は指定位置がスポーン可能かつ十分な広さに到達可能かを判定する
func (pf *PathFinder) isValidSpawnPosition(pos consts.Coord[consts.Tile]) bool {
	planData := pf.planData
	idx := planData.Level.CoordToIndex(pos)
	if int(idx) >= len(planData.Tiles) || planData.Tiles[idx].BlockPass {
		return false
	}

	// NPC・アイテム・ポータルなど計画済みエンティティとの重複を防ぐ
	if planData.existPlannedEntityOnTile(pos) {
		return false
	}

	// 隣接4方向のうち少なくとも1つは移動可能なタイルが必要。
	// 敵エンティティはBlockPassを持つため、全隣接タイルが占有されるとプレイヤーが移動不能になる
	if !pf.hasAdjacentFreeTile(pos) {
		return false
	}

	// 到達可能なタイル数が十分であることを確認する。
	// 単一BFSで済むため、複数BFSを要するポータル到達性チェックより先に行う
	if pf.countReachableFrom(pos) < minReachableTiles {
		return false
	}

	// NextPortalsへの到達性をチェック
	for _, portal := range planData.NextPortals {
		if !pf.IsReachable(pos, portal) {
			return false
		}
	}

	return true
}

// ValidatePortalReachability はプレイヤー開始位置から全ポータルへの到達性を検証する
func (pf *PathFinder) ValidatePortalReachability() error {
	// プレイヤー開始位置を取得
	playerPos, err := pf.FindPlayerStartPosition()
	if err != nil {
		return fmt.Errorf("%w: プレイヤー開始位置が設定されていません", ErrConnectivity)
	}

	// NextPortalsの到達性をチェック
	for i, portal := range pf.planData.NextPortals {
		if !pf.IsReachable(playerPos, portal) {
			return fmt.Errorf("%w: プレイヤー開始位置(%d,%d)からNextPortal[%d](%d,%d)への到達不可",
				ErrConnectivity, playerPos.X, playerPos.Y, i, portal.X, portal.Y)
		}
	}

	return nil
}
