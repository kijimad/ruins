package components

import "github.com/kijimaD/ruins/internal/oapi"

// ShelterType はタイルの囲われの度合い。屋内外の判定と、世界温度の受け方に効く。ゼロ値は屋外
type ShelterType string

// 囲われの度合い
const (
	ShelterNone    ShelterType = ""        // 屋外
	ShelterPartial ShelterType = "partial" // 半屋外。世界温度を中間の強さで受ける
	ShelterFull    ShelterType = "full"    // 屋内。世界温度を緩和して受ける
)

// TileEnvironment はタイルの環境属性を持つコンポーネント。囲われ・水・植生。
// 温度計算の入力になる。各要因を個別に保持し、ホバー時に内訳を表示できるようにする
type TileEnvironment struct {
	Shelter ShelterType
	Water   oapi.WaterExposure // ゼロ値は水の影響なし
	Foliage oapi.Foliage       // ゼロ値は植生なし
}

// TileEnvDefs は水と植生がタイルの気温に足す補正の表。読み込み時に raw から1度だけ組み立て、以後は変えない
type TileEnvDefs struct {
	Water   map[oapi.WaterExposure]int
	Foliage map[oapi.Foliage]int
}

// TempModifier は水と植生による加算℃の合計を返す。囲われは加算℃でなく緩和度なので含まない
func (d TileEnvDefs) TempModifier(env TileEnvironment) int {
	return d.Water[env.Water] + d.Foliage[env.Foliage]
}
