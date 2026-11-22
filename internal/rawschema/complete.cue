package rawschema

// 実データが揃っていることの検査。列挙の値ごとに行や欄の値をちょうど1つ持つ。欠けた値は必須のキーで、
// 列挙に無い値は閉じた struct で、重なった値は添字の衝突で落ちる。部分の fixture には求めないので #Raws から分け、
// rawcheck.ValidateComplete だけが検証する。値を列挙に足したら、ここの必須のキーにも足す
#Complete: {
	freshnessStages_complete: close({
		fresh!:  _
		stale!:  _
		rotten!: _
	}) & {if freshnessStages != _|_ for i, x in freshnessStages {(x.id): i}}

	timesOfDay_complete: close({
		dawn!:     _
		morning!:  _
		noon!:     _
		evening!:  _
		night!:    _
		midnight!: _
	}) & {if timesOfDay != _|_ for i, x in timesOfDay {(x.id): i}}

	seasons_complete: close({
		spring!: _
		summer!: _
		autumn!: _
		winter!: _
	}) & {if seasons != _|_ for i, x in seasons {(x.id): i}}

	pointFeatures_complete: close({
		settlement!: _
		landmark!:   _
	}) & {if pointFeatures != _|_ for i, x in pointFeatures {(x.id): i}}

	weathers_complete: close({
		clear!:     _
		cloudy!:    _
		rain!:      _
		snow!:      _
		blizzard!:  _
		cold_snap!: _
	}) & {if weathers != _|_ for i, x in weathers {(x.id): i}}

	// 天気の行の添字で束ねる。天気の id で束ねると、天気が重なったときに季節の報告が混ざる
	weatherSeasons_complete: {
		if weathers != _|_ for i, w in weathers {
			"\(i)": close({
				spring!: _
				summer!: _
				autumn!: _
				winter!: _
			}) & {for j, x in w.seasons {(x.id): j}}
		}
	}

	shelterEffects_complete: close({
		outdoor!: _
		partial!: _
		indoor!:  _
	}) & {if shelterEffects != _|_ for i, x in shelterEffects {(x.id): i}}

	waterEffects_complete: close({
		nearby!:    _
		submerged!: _
	}) & {if waterEffects != _|_ for i, x in waterEffects {(x.id): i}}

	hungerStages_complete: close({
		satiated!: _
		normal!:   _
		hungry!:   _
		starving!: _
	}) & {if hungerStages != _|_ for i, x in hungerStages {(x.id): i}}

	fatigueStages_complete: close({
		rested!:    _
		normal!:    _
		tired!:     _
		exhausted!: _
	}) & {if fatigueStages != _|_ for i, x in fatigueStages {(x.id): i}}

	proficiencies_complete: close({
		fire_resist!:      _
		thunder_resist!:   _
		chill_resist!:     _
		photon_resist!:    _
		cold_progress!:    _
		hunger_progress!:  _
		healing_effect!:   _
		max_weight!:       _
		exploration!:      _
		enemy_vision!:     _
		night_vision!:     _
		move_cost!:        _
		craft_cost!:       _
		smith_quality!:    _
		buy_price!:        _
		sell_price!:       _
		heavy_armor!:      _
		sword_damage!:     _
		spear_damage!:     _
		fist_damage!:      _
		bow_damage!:       _
		handgun_damage!:   _
		rifle_damage!:     _
		cannon_damage!:    _
		sword_accuracy!:   _
		spear_accuracy!:   _
		fist_accuracy!:    _
		bow_accuracy!:     _
		handgun_accuracy!: _
		rifle_accuracy!:   _
		cannon_accuracy!:  _
		throw_damage!:     _
		throw_accuracy!:   _
	}) & {
		if skills != _|_ for i, s in skills if s.proficiencies != _|_ for j, x in s.proficiencies {(x.id): "skills.\(i).proficiencies.\(j)"}
	}

	attackTypes_complete: close({
		fist!:    _
		sword!:   _
		spear!:   _
		handgun!: _
		rifle!:   _
		cannon!:  _
		bow!:     _
		throw!:   _
	}) & {if attackTypes != _|_ for i, x in attackTypes {(x.id): i}}

	conditionOnsets_complete: close({
		cold!:        _
		rotten_food!: _
		fatigue!:     _
		hunger!:      _
	}) & {if conditions != _|_ for i, x in conditions if x.onset != _|_ {(x.onset): i}}

	mapGlyphChunks_complete: close({
		wasteland!: _
	}) & {if mapGlyphs != _|_ for i, x in mapGlyphs if x.chunk != _|_ {(x.chunk): i}}

	// 無属性は軽減の対象外なので、無属性を除く属性を1つずつ軽減する
	resists_complete: close({
		FIRE!:    _
		THUNDER!: _
		CHILL!:   _
		PHOTON!:  _
	}) & {
		if skills != _|_ for i, s in skills if s.proficiencies != _|_ for j, x in s.proficiencies if x.resists != _|_ {(x.resists): "skills.\(i).proficiencies.\(j)"}
	}
} @rawgen(-)
