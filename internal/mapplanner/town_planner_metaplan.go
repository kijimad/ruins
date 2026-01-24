// Package mapplanner の MetaPlan 対応街プランナー
// 元の文字列ベース街レイアウトを MetaPlan 方式で実装
package mapplanner

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
)

// MetaTownPlanner は文字列ベース街マップを MetaPlan に直接構築するプランナー
type MetaTownPlanner struct {
	TileMap   []string
	EntityMap []string
}

// NewTownPlanner は MetaPlan 対応の街プランナーを作成する
func NewTownPlanner(width gc.Tile, height gc.Tile, seed uint64) (*PlannerChain, error) {
	// 50x50の街レイアウト（幅3の道路と5x5以上の建物）
	tileMap, entityMap := getTownLayout()

	// レイアウトの基本整合性を検証
	if err := validateTownLayout(tileMap, entityMap); err != nil {
		return nil, fmt.Errorf("街レイアウト検証エラー: %w", err)
	}

	planner := &MetaTownPlanner{
		TileMap:   tileMap,
		EntityMap: entityMap,
	}

	// 実際のマップサイズは文字列から自動検出される（50x50）
	chain := NewPlannerChain(width, height, seed)
	chain.StartWith(planner)
	return chain, nil
}

// PlanInitial は MetaPlan の初期化を行う
func (p *MetaTownPlanner) PlanInitial(planData *MetaPlan) error {
	// タイルマップのサイズを自動検出
	height := len(p.TileMap)
	if height == 0 {
		return fmt.Errorf("タイルマップが空です")
	}
	width := len(p.TileMap[0])

	// MetaPlan のサイズを更新
	planData.Level.TileWidth = gc.Tile(width)
	planData.Level.TileHeight = gc.Tile(height)

	// タイル配列を初期化
	totalTiles := width * height
	planData.Tiles = make([]raw.TileRaw, totalTiles)

	// 文字列マップからタイルを生成
	for y, row := range p.TileMap {
		for x, char := range row {
			idx := planData.Level.XYTileIndex(gc.Tile(x), gc.Tile(y))

			switch char {
			case '#':
				// 壁タイル
				planData.Tiles[idx] = planData.GetTile("Wall")
			case 'f':
				// 建物内の床タイル
				planData.Tiles[idx] = planData.GetTile("Floor")
			case 'r':
				// 道路タイル
				planData.Tiles[idx] = planData.GetTile("Floor")
			case 'd':
				// 土タイル（屋外の空き地）
				planData.Tiles[idx] = planData.GetTile("Dirt")
			default:
				return fmt.Errorf("無効なタイル指定子が存在する: %s", string(char))
			}
		}
	}

	// エンティティマップからNPC、アイテム、ワープポータルを配置
	for y, row := range p.EntityMap {
		if y >= len(p.EntityMap) {
			break
		}
		for x, char := range row {
			if x >= len(row) {
				break
			}

			switch char {
			case '.':
				// 何もしない
				continue
			case '@':
				// プレイヤー開始位置を設定
				planData.PlayerStartPosition = &struct {
					X int
					Y int
				}{X: x, Y: y}
				continue
			case '&':
				// NPC（街プランナーではNPCをスキップ - テスト用）
				// TODO: 適切なNPCタイプが定義されたら追加
				continue
			case '1':
				// 老兵NPC
				planData.NPCs = append(planData.NPCs, NPCSpec{
					X:    x,
					Y:    y,
					Name: "老兵",
				})
			case '$':
				// 商人NPC
				planData.NPCs = append(planData.NPCs, NPCSpec{
					X:    x,
					Y:    y,
					Name: "商人",
				})
			case '2':
				// 怪しい科学者NPC
				planData.NPCs = append(planData.NPCs, NPCSpec{
					X:    x,
					Y:    y,
					Name: "怪しい科学者",
				})
			case '3':
				// 闇医者NPC
				planData.NPCs = append(planData.NPCs, NPCSpec{
					X:    x,
					Y:    y,
					Name: "闇医者",
				})
			case 'w':
				// ワープホール
				planData.WarpPortals = append(planData.WarpPortals, WarpPortal{
					X:    x,
					Y:    y,
					Type: WarpPortalNext, // 次の階層への移動
				})
			case 'C', 'T', 'S', 'M', 'R', 'L', 'B', 'D':
				// Props
				var propKey string
				switch char {
				case 'C':
					propKey = "chair"
				case 'T':
					propKey = "table"
				case 'S':
					propKey = "bookshelf"
				case 'M':
					propKey = "barrel"
				case 'R':
					propKey = "crate"
				case 'L':
					propKey = "lantern"
				case 'B':
					propKey = "bonfire"
				case 'D':
					propKey = "door"
				}

				planData.Props = append(planData.Props, PropsSpec{
					X:    x,
					Y:    y,
					Name: propKey,
				})
			default:
				return fmt.Errorf("無効なエンティティ指定子が存在する: %s", string(char))
			}
		}
	}

	return nil
}

// getTownLayout は街のタイルとエンティティレイアウトを返す
func getTownLayout() ([]string, []string) {
	// 50x50の街レイアウト（幅3の道路と5x5以上の建物）
	tileMap := []string{
		"##################################################",
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##", // 北の境界道路（幅3）
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##",
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##",
		"#rrr#########f##########rrr#########f#########r###",
		"#rrr#fffffffffffffffffr#rrr#ffffffffffffffffff#r##", // 北区域の大きな建物
		"#rrr#fffffffffffffffffr#rrr#ffffffffffffffffff#r##",
		"#rrr#fffffffffffffffffr#rrr#ffffffffffffffffff#r##",
		"#rrr#fffffffffffffffffr#rrr#ffffffffffffffffff#r##",
		"#rrr#fffffffffffffffffr#rrr#ffffffffffffffffff#r##",
		"#rrr#########f##########rrr##########f#########r##",
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##", // 北から中央への道路（幅3）
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##",
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##",
		"#rrr####f###ddd####f###rrr####f###ddd####f####r###", // 住宅区域
		"#rrr#ffffff#ddd#ffffff#rrr#ffffff#ddd#fffffff#r###", // 5x5の家
		"#rrrfffffff#ddd#ffffff#rrr#ffffff#ddd#fffffff#r###",
		"#rrr#ffffff#dddfffffff#rrr#ffffff#ddd#fffffff#r###",
		"#rrr#ffffff#ddd#ffffff#rrrfffffff#ddd#fffffff#r###",
		"#rrr#ffffff#ddd#ffffff#rrr#ffffff#dddfffffffr#r###",
		"#rrr########ddd########rrr########ddd#########r###",
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##", // 中央大通り（幅3）
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##",
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##",
		"#rrr#########f#########rrr##########f#########r###", // 中央広場エリア
		"#rrr#fffffffffffffffrrfrrr#ffffffffffffffffff#r###", // 広場（床のみ、ワープホールはentityMapで定義）
		"#rrr#ffffffffffffffffrrrrr#ffffffffffffffffff#r###",
		"#rrr#ffffffffffffffffrrrrr#ffffffffffffffffff#r###",
		"#rrr#ffffffffffffffffrrrrr#ffffffffffffffffff#r###",
		"#rrr#ffffffffffffffffrrrrr#ffffffffffffffffff#r###",
		"#rrr#########f#########rrr##########f#########r###",
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##", // 広場から南への道路（幅3）
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##",
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##",
		"#rrr####f###rrr########rrr####f###rrr####f####r###", // 南区域の住宅
		"#rrr#ffffff#rrr#ffffff#rrr#ffffff#rrr#fffffff#r###", // 5x5の家
		"#rrrfffffff#rrr#ffffff#rrr#ffffff#rrr#fffffff#r###",
		"#rrr#ffffff#rrrfffffff#rrr#ffffff#rrr#fffffff#r###",
		"#rrr#ffffff#rrr#ffffff#rrrfffffff#rrr#fffffff#r###",
		"#rrr#ffffff#rrr#ffffff#rrr#ffffff#rrrfffffffr#r###",
		"#rrr########rrr######f#rrr########rrr#########r###",
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##", // 南区域の道路（幅3）
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##",
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##",
		"#rrrrrrrrrrrrrrrrrrrrrrrdrrrrrrrrrrrrrrrrrrrrrrr##",
		"#rrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrr##",
		"#rrrrrrrrrrrrrrrrrrrrrrdddrdrrrrrrrrrrrrrrrrrrrr##",
		"#rrrrrrrrrrrrrrrrrrrrrrdddrdrdddrrrrrrrrrrrrrrrr##",
		"#rrrrrrrrrrrrrrrrrrrrrrdddrdrrrrrrrrrrrrrrrrrrrr##",
		"##################################################",
	}

	// エンティティ配置（建物、Props、NPCなど） 50x50（幅3道路対応）
	entityMap := []string{
		"..................................................",
		"..................................................", // 北の境界道路（幅3）
		"..................................................",
		"..................................................",
		"..................................................",
		"....&.CT..........MS......SSSS....................", // 北区域の大きな建物
		"..................................................",
		"..................................................",
		"..................................................",
		"..MS.............................TTT..............",
		"..................................................",
		"..................................................", // 北から中央への道路（幅3）
		"..................................................",
		"..................................................",
		"..................................................", // 住宅区域
		"....&.TC.......&..TS......&..CT......&.S..........", // 5x5の家
		"..................................................",
		"....M.................................R...........",
		"..................................................",
		"..................................................",
		"..................................................",
		"..................................................", // 中央大通り（幅3）
		"..................................................",
		"..................................................",
		"..................................................", // 中央広場エリア
		"...........................&......................", // 広場（NPC）
		"..................................................", // プレイヤー開始位置
		"........................MM........................",
		"..................................................",
		"..................................................",
		"..................................................",
		"..................................................", // 広場から南への道路（幅3）
		"..................................................",
		"..................................................",
		"..................................................", // 南区域の住宅
		"....&..CT......&..........&ST.......&.R...........", // 5x5の家
		"..................................................",
		"...............D..................................",
		"..........................D.......................",
		"..................................................",
		".....................D............................",
		"...........L..........L...L..........L............", // 南区域の道路（幅3）
		"........................@.2.......................",
		"..................................................",
		"...................1.CT.B...$...3.................",
		"..................................................", // 広場
		"..................................................",
		"........................w.........................", // ワープホール（下の広場の真ん中）
		"..................................................",
		"..................................................",
	}

	return tileMap, entityMap
}

// validateTownLayout は街レイアウトの基本整合性を検証する（接続性を除く）
func validateTownLayout(tileMap, entityMap []string) error {
	if len(tileMap) == 0 || len(entityMap) == 0 {
		return fmt.Errorf("マップが空です")
	}

	height := len(tileMap)
	width := len(tileMap[0])

	// サイズ一致確認
	if len(entityMap) != height {
		return fmt.Errorf("タイルマップとエンティティマップの行数が一致しません: %d vs %d", len(tileMap), len(entityMap))
	}

	// 各行の長さ確認
	for i, row := range tileMap {
		if len(row) != width {
			return fmt.Errorf("タイルマップの行 %d の長さが不一致: 期待値 %d, 実際 %d", i, width, len(row))
		}
	}

	for i, row := range entityMap {
		if len(row) != width {
			return fmt.Errorf("エンティティマップの行 %d の長さが不一致: 期待値 %d, 実際 %d", i, width, len(row))
		}
	}

	// 境界壁の確認
	for x := 0; x < width; x++ {
		if tileMap[0][x] != '#' || tileMap[height-1][x] != '#' {
			return fmt.Errorf("上下の境界が壁でありません: 位置 (%d, 0) または (%d, %d)", x, x, height-1)
		}
	}

	for y := 0; y < height; y++ {
		if tileMap[y][0] != '#' || tileMap[y][width-1] != '#' {
			return fmt.Errorf("左右の境界が壁でありません: 位置 (0, %d) または (%d, %d)", y, width-1, y)
		}
	}

	// 有効なタイル文字の確認
	validTileChars := map[rune]bool{
		'#': true, // 壁
		'f': true, // 建物内の床
		'r': true, // 道路
		'd': true, // 土地（屋外）
		' ': true, // 空白（空のタイル）
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			char := rune(tileMap[y][x])
			if !validTileChars[char] {
				return fmt.Errorf("無効なタイル文字 '%c' が位置 (%d, %d) にあります", char, x, y)
			}
		}
	}

	// ワープホールの確認（tileMapとentityMapの両方をチェック）
	warpCount := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if tileMap[y][x] == 'w' {
				warpCount++
			}
		}
	}

	// entityMapでもワープホールをカウント
	for y := 0; y < height && y < len(entityMap); y++ {
		for x := 0; x < width && x < len(entityMap[y]); x++ {
			if entityMap[y][x] == 'w' {
				warpCount++
			}
		}
	}

	if warpCount != 1 {
		return fmt.Errorf("ワープホールが正確に1つある必要があります: 実際 %d", warpCount)
	}

	return nil
}
