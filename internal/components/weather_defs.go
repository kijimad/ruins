package components

// WeatherDef は天候1種の定義。raw の weathers の1行から組み立てる
type WeatherDef struct {
	Effect        WeatherEffect
	Severity      int        // 寒さの厳しさ。奥地ほど厳しい天候を厚くし、冬と奥地でスペルを長引かせる
	SpellTurnsMin int        // スペルの基準長の下限。ターン
	SpellTurnsMax int        // スペルの基準長の上限。ターン
	SeasonWeights [4]float64 // Season の並びごとの出やすさ
}

// WeatherDefs は天候の種類ごとの定義の表。読み込み時に raw から1度だけ組み立て、以後は変えない
type WeatherDefs struct {
	kinds [NumWeatherKind]WeatherDef
}

// NewWeatherDefs は天候の種類ごとの定義から表を作る
func NewWeatherDefs(kinds [NumWeatherKind]WeatherDef) WeatherDefs {
	return WeatherDefs{kinds: kinds}
}

// Def は天候の種類の定義を返す
func (d WeatherDefs) Def(k WeatherKind) WeatherDef {
	return d.kinds[k]
}
