package components

import "github.com/kijimaD/ruins/internal/oapi"

// TileEnvironment はタイルの環境属性を持つコンポーネント。囲われ・水・植生。
// 温度計算の入力になる。各要因を個別に保持し、ホバー時に内訳を表示できるようにする
type TileEnvironment struct {
	Shelter oapi.Shelter
	Water   oapi.WaterExposure // ゼロ値は水の影響なし
	Foliage oapi.Foliage       // ゼロ値は植生なし
}

// ShelterDef は囲われの度合いごとの、世界温度の受け方と表示名。raw の shelterEffects の1行から組み立てる
type ShelterDef struct {
	Name             string // 表示名 msgid
	AnchorTemp       int    // 世界温度を引き寄せる先の温度
	InfluencePercent int    // 世界温度を受ける割合。100 なら世界温度をそのまま受ける
}

// TileEnvDefs は囲われの受け方と、水と植生がタイルの気温に足す補正の表。読み込み時に raw から1度だけ組み立て、以後は変えない
type TileEnvDefs struct {
	Shelters map[oapi.Shelter]ShelterDef
	Water    map[oapi.WaterExposure]int
	Foliage  map[oapi.Foliage]int
}

// ShelteredWorldTemp は囲われに応じて受け方を変えた世界温度を返す。表に無い囲われは世界温度をそのまま受ける
func (d TileEnvDefs) ShelteredWorldTemp(shelter oapi.Shelter, worldTemp int) int {
	def, ok := d.Shelters[shelter]
	if !ok {
		return worldTemp
	}
	return def.AnchorTemp + (worldTemp-def.AnchorTemp)*def.InfluencePercent/100
}

// ShelterName は囲われの度合いの表示名 msgid を返す。表に無ければ id をそのまま返す
func (d TileEnvDefs) ShelterName(shelter oapi.Shelter) string {
	if def, ok := d.Shelters[shelter]; ok {
		return def.Name
	}
	return string(shelter)
}

// TempModifier は水と植生による加算℃の合計を返す。囲われは加算℃でなく緩和度なので含まない
func (d TileEnvDefs) TempModifier(env TileEnvironment) int {
	return d.Water[env.Water] + d.Foliage[env.Foliage]
}
