// Package rawschema は raw のデータの形、値の制約、行の間の参照を持つ。
package rawschema

import "strings"

// 能力値
#Abilities: {
	vitality!:  #Vitality
	strength!:  #Strength
	sensation!: #Sensation
	dexterity!: #Dexterity
	agility!:   #Agility
	defense!:   #Defense
}

// 能力値。スキルの成長の速さと熟練の効果倍率に効く
#AbilityId: "str" | "sen" | "dex" | "agi" | "vit" | "def"

// 命中率。0で必中なし、100で必中
#Accuracy: int & >=0 & <=100

// 命中率補正値
#AccuracyBonus: int & >=-100 & <=100

// 行動コスト
#ActionCost: int & >=0 & <=100

// 敏捷性。行動順に影響する
#Agility: int & >=-100 & <=100

// 環境光の色。明るさは daylight が持ち、ここは色味だけを表す
#AmbientColor: {
	r!: #LightLevel
	g!: #LightLevel
	b!: #LightLevel
}

// 弾薬設定
#Ammo: {
	ammoTag?:       #AmmoTag
	damageBonus!:   #DamageBonus
	accuracyBonus!: #AccuracyBonus
}

// 弾薬タグ
#AmmoTag: "9mm" | "rifle" | "shell" | "energy_cell" @rawgen(prefix)

// 防具の防御力
#ArmorDefense: int & >=0 & <=999

#AttackCategory: "fist" | "sword" | "spear" | "handgun" | "rifle" | "cannon" | "bow" @rawgen(prefix)

// 1ターンあたりの攻撃回数
#AttackCount: int & >=1 & <=10

// 攻撃種別。武器の攻撃種別は同じ名前の武器スキルで上達する攻撃の射程の区分
#AttackRange: "melee" | "ranged" @rawgen(prefix)

// 攻撃種別。射程などの計算は Go が持ち、raw は命中で負わせる怪我を持つ
#AttackType: {
	id!:    #AttackCategory
	name!:  #EntityName
	range!: #AttackRange

	// 遠距離だけが持つ。この距離までは命中が落ちない
	optimalRange?: #RangeTiles

	// 遠距離だけが持つ。この距離を超えると撃てない
	maxRange?: #RangeTiles

	// 遠距離だけが持つ。最適射程を1タイル超えるごとに下がる命中。百分率
	penaltyPerTile?: #RangePenaltyPercent
	injury!:         #ConditionTypeName
	injury_ref:      _conditionIDs[injury] @rawgen(-)

	// 攻撃で上達するスキル
	skill!: #SkillId

	// ダメージを決める熟練の効果倍率。skill の行が持つ
	damage!: #ProficiencyId

	// 命中を決める熟練の効果倍率。skill の行が持つ
	accuracy!: #ProficiencyId
}

// 基本ダメージ
#BaseDamage: int & >=0 & <=9999

// 寝具設定。眠るときの睡眠効率を決める
#Bedding: {
	quality!: #BeddingQuality
}

// 寝具の睡眠効率。100が基準で、高いほど短く深く眠れる
#BeddingQuality: int & >=0

// 通行を妨げるかどうか
#BlocksPassage: bool

// 視線を遮るかどうか
#BlocksView: bool

// 部位。部位の集合と身体機能との対応は Go が持ち、raw は表示名と命中の重みを持つ
#BodyPart: {
	id!:   #BodyPartId
	name!: #EntityName

	// 戦闘の命中抽選の重み。0 は命中先にならない
	hitWeight!: #EntryWeight
}

// 部位の id
#BodyPartId: string & strings.MinRunes(
		1) & strings.MaxRunes(
		50) & =~"^[a-z][a-z0-9_]*$"

// 本の設定
#Book: {
	totalEffort!: #ReadingEffort
	skill?:       #SkillBook
}

// 建物の散らかり段階
#BuildingClutter: "tidy" | "messy" | "filthy"

// 建物の損傷段階
#BuildingDamage: "intact" | "minor" | "major" @rawgen(prefix)

// 建物の事情。稀な1軒へ物語を与える
#BuildingVariant: "standard" | "abandoned" | "under_construction" | "survivalist" | "hoarder"

// 建物の散らかり段階の重み
#ClutterChoice: {
	id!:     #BuildingClutter
	weight!: #EntryWeight

	// 家具1つの隣へ小物を落とす割合。0 なら散らかりを足さない
	furniturePercent!: #InteriorPercent

	// 家具から離れた床のタイルへ小物を散らす割合
	floorPercent!: #InteriorPercent
}

// RGBA色チャネル値 (0-255)
#ColorChannel: int @rawgen(type=uint8)

// 戦闘ポリシー。エンティティの戦闘時の行動方針を定義する
#CombatPolicyType: "attack" | "evade" | "ignore"

// コマンドテーブル
#CommandTable: {
	id!:   #EntityID
	name!: #EntityName
	entries!: [...#CommandTableEntry]
}

// コマンドテーブルエントリ
#CommandTableEntry: {
	// 使う武器の id
	id!:     #EntityID
	id_ref:  _itemIDs[id] @rawgen(-)
	weight!: #EntryWeight
}

// 不調。痛みと身体機能の低下、回復の仕方、出血を持つ
#Condition: {
	id!:                      #ConditionTypeName
	name!:                    #EntityName
	description!:             #EntityDescription
	painPerSeverity!:         #ConditionAmountPerSeverity
	bodyFuncDropPerSeverity!: #ConditionAmountPerSeverity

	// 省くと ConditionSystem の管轄外になる
	recovery?: #RecoveryMode

	// progress_until_tend で未治療のとき1ターンに進行度を増やす量
	worsenPer?: #ConditionTimerRate

	// 治療済みで1ターンに進行度を減らす基準量
	recoverPer?:           #ConditionTimerRate
	bloodDropPerSeverity?: #ConditionAmountPerSeverity
	cause?:                #ConditionDeathCause

	// 起こりうる部位。省くと全部位
	parts?: [...#BodyPartId]
	if parts != _|_ {
		parts_ref: [for x in parts {_bodyPartIDs[x]}] @rawgen(-)
	}

	// 不調を起こす仕組み。攻撃で負う怪我と、きっかけの無い不調は省く
	onset?: #ConditionOnset
}

// 不調の重症度1段あたりの量。痛み、身体機能の低下、血液量の低下に使う
#ConditionAmountPerSeverity: int & >=0 & <=100

// 不調が血液量を尽きさせて倒したときの死因
#ConditionDeathCause: "frozen" | "illness" | "blood_loss"

// 不調を起こす仕組み。仕組みごとにちょうど1つの不調が持つ
#ConditionOnset: "cold" | "rotten_food" | "fatigue" | "hunger"

// 不調の重症度
#ConditionSeverity: "minor" | "medium" | "severe" @rawgen(prefix)

// 不調の1ターンあたりの進行度の増減量
#ConditionTimerRate: int & >=1 & <=100

// 不調の id。Go の ConditionType 定数の値と同じ
#ConditionTypeName: string & strings.MinRunes(
			1) & strings.MaxRunes(
			50) & =~"^[a-z][a-z0-9_]*$"

// 消費可能アイテムの設定
#Consumable: {
	usableScene!: #UsableScene
	targetGroup!: #TargetGroup
	targetNum!:   #TargetNum
}

// 抽選単位の束
#ContentGroup: {
	style!: #GroupStyle

	// pick_n のときの選ぶ個数。pick_each / pick_one では使わないので省略可
	pick?: int32
	items!: [...#ContentStuff]
}

// anchor 相対に一緒に置く衛星。机に対する椅子など。束で置いて散布事故を防ぐ
#ContentSatellite: {
	kind!: #StuffKind
	ref!:  #EntityID
	offsets!: [...#ContentVec]

	// 収納家具の中身を引く itemTables の id。収納家具では必須
	lootTable?: #EntityID
	if lootTable != _|_ {
		lootTable_ref: _itemTableIDs[lootTable] @rawgen(-)
	}
}

// 内装レシピの1配置指示
#ContentStuff: {
	kind!:   #StuffKind
	ref!:    #EntityID
	weight?: #EntryWeight

	// 置く個数のダイス表記
	amount!:    #Dice
	placement?: #Placement
	satellites?: [...#ContentSatellite]

	// 収納家具の中身を引く itemTables の id。収納家具では必須
	lootTable?: #EntityID
	if lootTable != _|_ {
		lootTable_ref: _itemTableIDs[lootTable] @rawgen(-)
	}
}

// 相対座標。衛星の anchor からのオフセットに使う。負値は上/左方向
#ContentVec: {
	x!: int32
	y!: int32
}

// キューブモジュール設定。装着すると展開野営の範囲や燃料タンクの容量を広げる
#CubeModule: {
	rangeBonus!:        #RangeBonus
	fuelCapacityBonus!: #FuelCapacityBonus
}

// ダメージ補正値
#DamageBonus: int & >=-100 & <=100

// 建物の損傷段階の重み
#DamageChoice: {
	id!:     #BuildingDamage
	weight!: #EntryWeight

	// 経年で戦利品を撤去する割合
	lootPercent!: #InteriorPercent

	// 廃墟化で壁際のタイルに瓦礫の種を撒く割合。0 から 1
	rubbleSeedRatio!: #NeedRatio
}

// 危険度。経過日数から決まる
#DangerLevel: int & >=0 & <=100

// 防御力。被ダメージを軽減する
#Defense: int & >=-100 & <=100

// 建物の家具量の倍率の重み
#DensityChoice: {
	factor!: #DensityFactor
	weight!: #EntryWeight
}

// 建物の家具量の倍率。10 が等倍の ×/10
#DensityFactor: int & >=1 & <=30

// 装着設定。キューブの展開空間へ装着するとフィールドで設備として働くアイテムであることを表す
#Deployable: {
	storage?:     #StorageRaw
	lightSource?: #LightSource
	bedding?:     #Bedding
}

// 器用さ。クリティカル率に影響する
#Dexterity: int & >=-100 & <=100

// 会話データ
#Dialog: {
	messageKey!:    #MessageKey
	messageKey_ref: _dialogueIDs[messageKey] @rawgen(-)
}

// NPC の会話。member の dialog の messageKey が id を指す。ページの順に1ページずつ見せる
#Dialogue: {
	id!: #MessageKey
	pages!: [...#DialoguePage] & [_, ...]
}

// 会話の1ページ。英語の msgid で、表示のとき query.T で訳す。<keyword> などのマークアップを含められる
#DialoguePage: string & strings.MinRunes(
		1)

// ダイス表記の個数指定。固定値も 1d1 のように書く。consts.ParseDice でパースする
#Dice: string & =~"^[1-9][0-9]*d[1-9][0-9]*([+-][0-9]+)?$"

// 分解定義。これを持つ prop は殴って壊すと産出を落とす。toolCategory の道具を構えて壊すと多く得る
#Disassembly: {
	toolCategory?: #ToolCategory
	toolRequired?: #DisassemblyToolRequired
	yields!: [...#DisassemblyYield]
	bonus?: [...#DisassemblyBonus]
}

// 分解のボーナス産出。minSkill か minGrade の少なくとも一方を指定する。両方指定した場合は両方を満たす必要がある
#DisassemblyBonus: {
	id!:    #EntityID
	id_ref: _itemIDs[id] @rawgen(-)

	// 産出個数のダイス表記
	count!: #Dice

	// 機械スキルがこの値以上で産出に加わる
	minSkill?: #SkillLevel

	// 工具グレードがこの値以上で産出に加わる
	minGrade?: #ToolGrade
}

// 分解産出の確率。百分率
#DisassemblyChance: int & >=1 & <=100

// 分解定義の道具が無いと殴っても削れないか
#DisassemblyToolRequired: bool

// 分解の産出エントリ。chance 省略は確定枠
#DisassemblyYield: {
	id!:    #EntityID
	id_ref: _itemIDs[id] @rawgen(-)

	// 産出個数のダイス表記
	count!:  #Dice
	chance?: #DisassemblyChance
}

// 扉ローデータ
#DoorRaw: {
	horizontalOpen!:   #SpriteKey
	horizontalClosed!: #SpriteKey
	verticalOpen!:     #SpriteKey
	verticalClosed!:   #SpriteKey
}

// ランドマークの描画関数の選択キー
#DrawerKey: "hut" | "open"

// ドロップテーブル
#DropTable: {
	id!:   #EntityID
	name!: #EntityName

	// 何も落とさない重み。省略すると0で、必ずどれかを落とす
	nothing?: #EntryWeight
	entries!: [...#DropTableEntry]
}

// ドロップテーブルエントリ
#DropTableEntry: {
	// ドロップする素材の id
	id!:     #EntityID
	id_ref:  _itemIDs[id] @rawgen(-)
	weight!: #EntryWeight
}

// フロアを生成して潜るダンジョン。id はステージの識別名で、遺跡の入口が指す
#Dungeon: {
	id!:          #EntityID
	name!:        #EntityName
	description!: #EntityDescription

	// 背景画像のスプライトキー
	imageKey!: #EntityID

	// 敵を引く enemyTables の id
	enemyTable!:    #EntityID
	enemyTable_ref: _enemyTableIDs[enemyTable] @rawgen(-)

	// アイテムを引く itemTables の id
	itemTable!:    #EntityID
	itemTable_ref: _itemTableIDs[itemTable] @rawgen(-)
	baseTemp!:     #StageBaseTemp

	// フロアの間取りの抽選表。行の順に重みを積む
	planners!: [...#DungeonPlannerWeight] & [_, ...]
}

// ダンジョンのフロアを生成する間取りの種類
#DungeonPlanner: "small_room" | "big_room" | "cave" | "ruins" | "forest" @rawgen(prefix)

// ダンジョンの間取りの種類と、フロアごとに選ばれる重み
#DungeonPlannerWeight: {
	id!:     #DungeonPlanner
	weight!: #EntryWeight
}

// 攻撃属性
#Element: "NONE" | "FIRE" | "THUNDER" | "CHILL" | "PHOTON" @rawgen(prefix)

// 敵テーブル
#EnemyTable: {
	id!:   #EntityID
	name!: #EntityName
	entries!: [...#EnemyTableEntry]
}

// 敵テーブルエントリ
#EnemyTableEntry: {
	id!:        #EntityID
	id_ref:     _memberIDs[id] @rawgen(-)
	weight!:    #EntryWeight
	minDanger!: #DangerLevel
	maxDanger!: #DangerLevel

	// 1群あたりの敵数のダイス表記
	pack!: #Dice
}

// 説明文
#EntityDescription: string & strings.MinRunes(
			1) & strings.MaxRunes(
			200)

#EntityID: string & strings.MinRunes(
		1) & strings.MaxRunes(
		50)

// エンティティ名
#EntityName: string & strings.MinRunes(
		1) & strings.MaxRunes(
		50)

// テーブルエントリの重み。大きいほど選ばれやすい
#EntryWeight: number & >=0

// 装備ボーナス
#EquipBonus: {
	vitality!:  #Vitality
	strength!:  #Strength
	sensation!: #Sensation
	dexterity!: #Dexterity
	agility!:   #Agility
}

// 装備スロット。防具部位 + 武器スロット
#EquipSlot: "HEAD" | "TORSO" | "ARMS" | "HANDS" | "LEGS" | "FEET" | "WEAPON1" | "WEAPON2" | "WEAPON3" | "WEAPON4" | "WEAPON5" @rawgen(prefix)

// 装備部位
#EquipmentCategory: "HEAD" | "TORSO" | "ARMS" | "HANDS" | "LEGS" | "FEET" | "JEWELRY" @rawgen(prefix)

// 街路側の前壁に付ける外皮の interiorRefs の id
#FacadeRefs: {
	window!:    #EntityID
	window_ref: _propRefIDs[window] @rawgen(-)

	// 大破した店が窓の代わりに下ろすシャッター
	shutter!:    #EntityID
	shutter_ref: _propRefIDs[shutter] @rawgen(-)

	// 店の入口脇に立てる看板
	sign!:    #EntityID
	sign_ref: _propRefIDs[sign] @rawgen(-)
}

// 市街地の施設1種。isShop は看板やシャッターを出す店か
#Facility: {
	id!:            #EntityID
	isShop!:        bool
	planner!:       #PlannerKey
	enemyTable!:    #EntityID
	enemyTable_ref: _enemyTableIDs[enemyTable] @rawgen(-)
	zones!: [...#FacilityZone]

	// 収納家具が荒らされて空になっている確率。0から100の百分率で、省略は0
	lootedChance?: int32 & >=0 & <=100
}

// 施設種別ごとの主室の内装変種。抽選で1つ選ぶ
#FacilityContent: {
	facility!:    #EntityID
	facility_ref: _facilityIDs[facility] @rawgen(-)
	variants!: [...#EntityID]
	variants_ref: [for x in variants {_interiorContentIDs[x]}] @rawgen(-)
}

// 施設種別ごとの奥室カタログ。役割別 content と、カタログに無い役割のフォールバック
#FacilityRooms: {
	facility!:    #EntityID
	facility_ref: _facilityIDs[facility] @rawgen(-)
	rooms?: [...#RoomContent]
	fallback!:    #EntityID
	fallback_ref: _interiorContentIDs[fallback] @rawgen(-)
}

// 施設が出現する地区と抽選重み。市街地の一辺が minSpan 以上のときだけ抽選対象になる
#FacilityZone: {
	zone!:    #Zone
	weight!:  int32 & >=1
	minSpan!: int32
}

// 派閥タイプ
#FactionMemberType: "FactionNeutral"

// 疲労の段階。疲労の割合が from 以上の段階のうち from が最大のものになる
#FatigueStage: {
	id!:   #FatigueStageId
	name!: #EntityName
	from!: #NeedRatio

	// この段階で眠れるか。疲れが足りない段階は眠れない
	canSleep!: bool

	// HUD に出すバッジの色。省くとバッジを出さない
	badgeColor?: #RGBAColor

	// この段階で立つ過労の重症度。省くと立たない
	severity?: #ConditionSeverity
}

// 疲労の段階
#FatigueStageId: "rested" | "normal" | "tired" | "exhausted" @rawgen(prefix)

// 遠距離攻撃設定
#Fire: {
	accuracy!:       #Accuracy
	damage!:         #BaseDamage
	attackCount!:    #AttackCount
	element!:        #Element
	attackCategory!: #AttackCategory
	cost!:           #ActionCost
	targetGroup!:    #TargetGroup
	targetNum!:      #TargetNum
	magazineSize!:   #MagazineSize
	reloadEffort!:   #ReloadEffort
	ammoTag?:        #AmmoTag
}

// 火種設定。所持していると隣接タイルの燃焼物に火をつけられる道具を表す
#FireStarter: {}

// タイルの植生
#Foliage: "grass" | "forest" @rawgen(prefix)

// 植生がタイルの気温に足す補正
#FoliageEffect: {
	id!:           #Foliage
	tempModifier!: #TileTempModifier
}

// 鮮度の段階の境目と表示名と、食べたとき得る栄養の割合
#FreshnessStage: {
	id!:   #FreshnessStageId
	name!: #EntityName

	// この段階が始まる劣化量。食料の stageLength の何倍かで表す。経過した段階数以下で from が最大の段階になり、from の小さい順が新しいものからの並び順になる
	from!:             #FreshnessStageFrom
	nutritionPercent!: #NutritionPercent

	// 食べたときのログの msgid。%s は食べた者と食べ物
	eatLog!: #EntityDescription

	// メニューの行に鮮度を添えるか
	marker!: bool

	// 食べたとき食中毒を起こす確率。百分率。省くと起こさない
	illnessChancePercent?: #IllnessChancePercent
}

// 鮮度の段階が始まる劣化量。食料の stageLength の何倍か
#FreshnessStageFrom: int & >=0

// 鮮度の段階
#FreshnessStageId: "fresh" | "stale" | "rotten"

// キューブモジュールの燃料容量ボーナス。装着するとキューブの燃料ポイントの上限がこの量だけ増える
#FuelCapacityBonus: int & >=0

// 持ち主の行を持たないチャンクの種類。地図記号の行がどの種類を表すかを指す
#GlyphChunk: "wasteland" | "dungeon_entrance"

// 地面を覆う草や土の飾りの prop 1種と、選ばれる重み
#GroundCoverEntry: {
	ref!:    #EntityID
	weight!: int32 & >=1
}

// グループの抽選方式。保証セットとランダム充填を分ける
#GroupStyle: "pick_each" | "pick_one" | "pick_n"

// 回復固定量
#HealAmount: int & >=0 & <=9999

// 回復割合。0.0〜1.0
#HealRatio: number & >=0 & <=1

// 回復量の計算方式
#HealingValueType: "PERCENTAGE" | "ABSOLUTE" | "NUMERAL"

// 材質のkgあたり燃焼熱量。0 なら不燃で燃料にならない
#HeatPerKg: int & >=0 & <=10000

// 熱の到達半径。チェビシェフ距離、タイル単位
#HeatRadius: int & >=1 & <=100

// 熱源設定。近接するキャラの低体温を毎ターン回復する
#HeatSource: {
	radius!: #HeatRadius
	warmth!: #HeatWarmth
}

// 熱源が毎ターン下げる低体温タイマーの量
#HeatWarmth: number & >=0

// hero 部屋の目玉の種類の重み。選ばれた種類の refs から1つを一様に引く
#HeroChoice: {
	id!:     #EntityID
	weight!: #EntryWeight
	refs!: [...#EntityID] & [_, ...]
	refs_ref: [for x in refs {_propRefIDs[x]}] @rawgen(-)
}

// 耐久値。削り切られると壊れる
#HitPoints: int & >=1 & <=9999

// 空腹の段階。満腹度の割合が from 以上の段階のうち from が最大のものになる
#HungerStage: {
	id!:   #HungerStageId
	name!: #EntityName
	from!: #NeedRatio

	// 食べてこの段階に達したときのログの msgid。%s は食べた者。省くと出さない
	reachedLog?: #EntityDescription

	// 寝ている者を起こす段階か
	wakesSleeper?: bool

	// HUD に出すバッジの色。省くとバッジを出さない
	badgeColor?: #RGBAColor

	// この段階で立つ栄養失調の重症度。省くと立たない
	severity?: #ConditionSeverity
}

// 空腹の段階
#HungerStageId: "satiated" | "normal" | "hungry" | "starving" @rawgen(prefix)

// 病気を起こす確率。百分率
#IllnessChancePercent: int & >=0 & <=100

// 画像ファイルパス
#ImagePath: string & strings.MinRunes(
		1) & strings.MaxRunes(
		200) & =~"^[a-zA-Z0-9_/\\-]+\\.[a-zA-Z0-9]+$"

// 耐寒性能
#InsulationCold: int & >=-100 & <=100

// 耐暑性能
#InsulationHeat: int & >=-100 & <=100

// 内装レシピ。施設まるごと、または奥室1つに対応する
#InteriorContent: {
	id!: #EntityID
	groups?: [...#ContentGroup]
}

// 内装の経年や散らかりの割合。百分率
#InteriorPercent: int & >=0 & <=100

// 建物単位の生活感の抽選表。建物ごとに各軸を1回引き、全室へ一様に効かせる
#InteriorProfile: {
	damages!: [...#DamageChoice]
	clutters!: [...#ClutterChoice]
	variants!: [...#VariantChoice]
	densities!: [...#DensityChoice]
	heroes!: [...#HeroChoice]
	heroIds_unique: {if heroes != _|_ for i, x in heroes {(x.id): i}} @rawgen(-)
	sites!: [...#SiteChoice]
	facade!: #FacadeRefs
	lot!:    #LotRefs

	// 経年で家具1つが隣へ生活痕を落とす割合
	wearPercent!: #InteriorPercent

	// hero 部屋の目玉を持つ建物の割合の逆数。この数の建物に1棟が目玉を持つ
	heroOneIn!: int32 & >=1
}

// 内装レシピの Ref。spawn する prop か床 loot の item group と、既定の置き方を持つ
#InteriorRef: {
	id!: #EntityID

	// 家具と装飾として置く raw の prop。省くと prop として置かない
	prop?: #EntityID
	if prop != _|_ {
		prop_ref: _propIDs[prop] @rawgen(-)
	}

	// 床 loot として引く itemGroups か itemCollections の id。省くと床 loot として置かない
	itemGroup?: #EntityID
	if itemGroup != _|_ {
		itemGroup_ref: _itemGroupIDs[itemGroup] @rawgen(-)
	}

	// レシピが placement を省いたときの置き方
	placement?: #Placement
}

// プレイヤーキャラクターかどうか
#IsPlayer: bool

// アイテム
#Item: {
	id!:              #EntityID
	name!:            #EntityName
	description!:     #EntityDescription
	spriteSheetName!: #SpriteSheetName
	spriteKey!:       #SpriteKey
	animKeys?: [...#SpriteKey]
	value!: #ItemValue

	// アイテム重量
	weight?:            #Weight
	inflictsDamage?:    #BaseDamage
	providesNutrition?: #NutritionAmount
	stageLength?:       #StageLengthTurns
	consumable?:        #Consumable
	providesHealing?:   #ProvidesHealing
	remedy?:            #Remedy
	wearable?:          #Wearable
	equipBonus?:        #EquipBonus
	ammo?:              #Ammo
	cubeModule?:        #CubeModule
	deployable?:        #Deployable
	melee?:             #Melee
	fire?:              #Fire
	book?:              #Book

	// 道具の性質。合う物を殴って壊すと速く多く得る
	tool?: #Tool

	// 携行光源。装備すると owner を照らす
	lightSource?: #LightSource

	// 材質。可燃性と燃焼熱量を導く。可燃な材質のアイテムは地面に置いて着火したり火にくべて燃やせる
	material?: #Material

	// 火種。所持していると隣接の燃焼物に着火できる
	fireStarter?: #FireStarter
}

// アイテムコレクション。各エントリを chance の確率で独立に判定する。複数当選することもありうる
#ItemCollection: {
	id!:   #EntityID
	name!: #EntityName
	entries!: [...#ItemCollectionEntry]
}

// アイテムコレクションのエントリの出現確率。百分率。書き忘れて0になった行を弾くため0は許さない
#ItemCollectionChance: number & <=100 & >0

// アイテムコレクションエントリ
#ItemCollectionEntry: {
	id!:     #EntityID
	id_ref:  _itemIDs[id] @rawgen(-)
	chance!: #ItemCollectionChance

	// 1山あたりの個数のダイス表記
	pack!: #Dice
}

// アイテム所持数
#ItemCount: int & >=1 & <=9999

// アイテムグループ。エントリ群から weight の比で1つだけ選ぶ
#ItemGroup: {
	id!:   #EntityID
	name!: #EntityName
	entries!: [...#ItemGroupEntry]
}

// アイテムグループエントリ
#ItemGroupEntry: {
	id!:     #EntityID
	id_ref:  _itemIDs[id] @rawgen(-)
	weight!: #EntryWeight

	// 1山あたりの個数のダイス表記
	pack!: #Dice
}

// アイテムテーブル
#ItemTable: {
	id!:   #EntityID
	name!: #EntityName
	entries!: [...#ItemTableEntry]
}

// アイテムテーブルエントリ。アイテムグループかアイテムコレクションを参照する
#ItemTableEntry: {
	// 参照するアイテムグループかアイテムコレクションの id
	id!:        #EntityID
	id_ref:     _itemGroupIDs[id] @rawgen(-)
	weight!:    #EntryWeight
	minDanger!: #DangerLevel
	maxDanger!: #DangerLevel
}

// 売買価格
#ItemValue: int & >=0 & <=99999

// 原野の点在ランドマーク1種。hutW/hutH は drawer=hut のときだけ使う
#Landmark: {
	id!:     #EntityID
	weight!: int32 & >=1
	drawer!: #DrawerKey
	hutW!:   int32
	hutH!:   int32
	props!: [...#PropSpot]
	props_ref: [for x in props {_propIDs[x.name]}] @rawgen(-)
}

// 光源が有効かどうか
#LightEnabled: bool

// 光の強さ。0 が真っ暗、1 が最大
#LightLevel: number & >=0 & <=1

// 光の到達半径（タイル単位）
#LightRadius: int & >=1 & <=100

// 光源設定
#LightSource: {
	radius!:  #LightRadius
	color!:   #RGBAColor
	enabled!: #LightEnabled
}

// 敷地の外構の interiorRefs の id
#LotRefs: {
	fence!:    #EntityID
	fence_ref: _propRefIDs[fence] @rawgen(-)

	// 民家の前庭に置く外構
	plant!:    #EntityID
	plant_ref: _propRefIDs[plant] @rawgen(-)

	// 店の前庭に置く外構
	vending!:    #EntityID
	vending_ref: _propRefIDs[vending] @rawgen(-)
}

// マガジン容量
#MagazineSize: int & >=1 & <=100

// 俯瞰地図の記号1種。id は地物・施設の種別 id
#MapGlyph: {
	id!:    #EntityID
	glyph!: string
	name!:  string
	color!: #RGBAColor
	order!: int32

	// この記号で描く、持ち主の行を持たないチャンクの種類。施設と集落とランドマークの記号は持ち主の行の id で引くので省く
	chunk?: #GlyphChunk
}

// 材質。可燃性と燃焼熱量の算出に使う。燃料熱量は材質のkgあたり熱量へ重量を掛けて導く。
// 不燃の材質は係数0で燃料にならない。係数は materials の行が持つ
#Material: "WOOD" | "PAPER" | "CLOTH" | "LEATHER" | "PLANT" | "FOOD" | "BONE" | "OIL" | "COAL" | "PLASTIC" | "METAL" | "STONE" | "GLASS" | "CRYSTAL" | "CERAMIC" | "LIQUID"

// 素材必要数
#MaterialAmount: int & >=1 & <=99

// 材質の表示名と燃焼の熱量
#MaterialDef: {
	id!:        #Material
	name!:      #EntityName
	heatPerKg!: #HeatPerKg
}

// 近接攻撃設定
#Melee: {
	accuracy!:       #Accuracy
	damage!:         #BaseDamage
	attackCount!:    #AttackCount
	element!:        #Element
	attackCategory!: #AttackCategory
	cost!:           #ActionCost
	targetGroup!:    #TargetGroup
	targetNum!:      #TargetNum
}

// メンバー
#Member: {
	id!:              #EntityID
	name!:            #EntityName
	player?:          #IsPlayer
	abilities!:       #Abilities
	spriteSheetName!: #SpriteSheetName
	spriteKey!:       #SpriteKey
	animKeys?: [...#SpriteKey]
	lightSource?:     #LightSource
	factionType?:     #FactionMemberType
	combatPolicy?:    #CombatPolicyType
	movementPattern?: #MovementPatternType
	viewDistance?:    #ViewDistance
	dialog?:          #Dialog

	// プレイヤーキャラクターでは省略可能
	commandTableId?: #EntityID
	if commandTableId != _|_ {
		commandTableId_ref: _commandTableIDs[commandTableId] @rawgen(-)
	}

	// プレイヤーキャラクターでは省略可能
	dropTableId?: #EntityID
	if dropTableId != _|_ {
		dropTableId_ref: _dropTableIDs[dropTableId] @rawgen(-)
	}

	// 生成時に装備して持たせるアイテム。startingItems より先に持たせる
	startingEquips?: [...#ProfessionEquip]
	if startingEquips != _|_ {
		startingEquips_ref: [for x in startingEquips {_itemIDs[x.name]}] @rawgen(-)
	}

	// 生成時にバックパックへ持たせるアイテム
	startingItems?: [...#ProfessionItem]
	if startingItems != _|_ {
		startingItems_ref: [for x in startingItems {_itemIDs[x.name]}] @rawgen(-)
	}

	// 店の在庫。持つ member は店を開く
	stock?: [...#ProfessionItem]
	if stock != _|_ {
		stock_ref: [for x in stock {_itemIDs[x.name]}] @rawgen(-)
	}
}

// メッセージリソースのキー
#MessageKey: string & strings.MinRunes(
		1) & strings.MaxRunes(
		100) & =~"^[a-z][a-z0-9_]*$"

// 非戦闘時の移動パターン
#MovementPatternType: "random" | "patrol" | "wallHug" | "stationary" | "wander" | "territorial" | "swarm" | "follow"

// 最大値に対する割合
#NeedRatio: number & >=0 & <=1

// 栄養価
#NutritionAmount: int & >=0 & <=9999

// 食料から得る栄養の満額に対する割合。百分率
#NutritionPercent: int & >=0 & <=100

// 通行コスト加算値。0で変化なし、50でベースコスト+50
#PassCost: int & >=0 & <=9999

// 配置の置き方。空なら interiorRefs の既定、それも無ければ全域へ落ちる
#Placement: "center" | "wall" | "full_area" | "near_door" | "far_from_door" | "row"

// 間取りテンプレの選択キー
#PlannerKey: "house" | "store" | "clinic" | "bsp"

// 治療の質。基準100の倍率。100が標準、150で回復1.5倍
#Potency: int & >=0 & <=9999

// 職業
#Profession: {
	id!:          #ProfessionId
	name!:        #EntityName
	description!: #EntityDescription
	abilities!:   #Abilities
	skills?: [...#ProfessionSkill]
	items!: [...#ProfessionItem]
	equips!: [...#ProfessionEquip]
}

// 職業初期装備
#ProfessionEquip: {
	name!: #EntityName
	slot!: #EquipSlot
}

// 職業ID
#ProfessionId: string & strings.MinRunes(
		1) & strings.MaxRunes(
		50) & =~"^[a-z][a-z0-9_]*$"

// 職業初期所持アイテム
#ProfessionItem: {
	name!:  #EntityName
	count!: #ItemCount
}

// 職業スキル初期値
#ProfessionSkill: {
	id!:    #SkillId
	id_ref: _skillIDs[id] @rawgen(-)
	value!: #SkillLevel
}

// 熟練の効果倍率の id。技能と能力値から導く行動ごとの倍率を指す。怪我・病気・疲労・空腹による身体機能の低下は別に扱う
#ProficiencyId: "fire_resist" | "thunder_resist" | "chill_resist" | "photon_resist" | "cold_progress" | "hunger_progress" | "healing_effect" | "max_weight" | "exploration" | "enemy_vision" | "night_vision" | "move_cost" | "craft_cost" | "smith_quality" | "buy_price" | "sell_price" | "heavy_armor" | "sword_damage" | "spear_damage" | "fist_damage" | "bow_damage" | "handgun_damage" | "rifle_damage" | "cannon_damage" | "sword_accuracy" | "spear_accuracy" | "fist_accuracy" | "bow_accuracy" | "handgun_accuracy" | "rifle_accuracy" | "cannon_accuracy" @rawgen(prefix)

// 熟練の効果倍率のスキルLv1あたりの変化量。百分率
#ProficiencyPerLevel: int & >=-100 & <=100

// 置物
#Prop: {
	id!:           #EntityID
	name!:         #EntityName
	description!:  #EntityDescription
	spriteRender!: #SpriteRender
	animKeys?: [...#SpriteKey]
	blockPass!: #BlocksPassage
	blockView!: #BlocksView
	passCost?:  #PassCost

	// 耐久。省略した prop は殴れない
	hp?:              #HitPoints
	lightSource?:     #LightSource
	heatSource?:      #HeatSource
	bedding?:         #Bedding
	door?:            #DoorRaw
	warpNextTrigger?: #WarpNextTriggerRaw
	warpPrevTrigger?: #WarpPrevTriggerRaw
	storage?:         #StorageRaw
	disassembly?:     #Disassembly

	// 燃え尽きたときに残すアイテム。火のように燃え尽きる prop だけが持つ
	burnoutItem?: #EntityID
	if burnoutItem != _|_ {
		burnoutItem_ref: _itemIDs[burnoutItem] @rawgen(-)
	}
}

// 基準点から相対の位置に置くエンティティ1個
#PropSpot: {
	name!: #EntityID
	dx!:   int32
	dy!:   int32
}

// 回復効果
#ProvidesHealing: {
	valueType!: #HealingValueType
	amount!:    #HealAmount
	ratio!:     #HealRatio
}

// RGBA色
#RGBAColor: {
	r!: #ColorChannel
	g!: #ColorChannel
	b!: #ColorChannel
	a!: #ColorChannel
}

// キューブ展開範囲の伸び幅。縦横を一律に広げるタイル数
#RangeBonus: int & >=0 & <=20

// 射程の超過1タイルあたりの命中の低下。百分率
#RangePenaltyPercent: int & >=0 & <=100

// 射程。タイル
#RangeTiles: int & >=1 & <=50

// ローデータ全体。TOMLファイルのルート構造を定義する
#Raws: {
	items?: [...#Item]
	recipes?: [...#Recipe]
	members?: [...#Member]
	commandTables?: [...#CommandTable]
	dropTables?: [...#DropTable]
	itemGroups?: [...#ItemGroup]
	itemCollections?: [...#ItemCollection]
	itemTables?: [...#ItemTable]
	enemyTables?: [...#EnemyTable]
	facilities?: [...#Facility]
	landmarks?: [...#Landmark]
	scatterZones?: [...#ScatterZone]
	mapGlyphs?: [...#MapGlyph]
	interiorContents?: [...#InteriorContent]
	facilityContents?: [...#FacilityContent]
	facilityRooms?: [...#FacilityRooms]

	// 全施設共通のフレーバー装飾。施設レシピと直交し全室へ一様に重ねる。id は直接引かないので
	// interiorContents プールには入れず、この専用フィールドだけに置く
	flavorContent?: #InteriorContent
	spriteSheets?: [...#SpriteSheet]
	tiles?: [...#Tile]
	props?: [...#Prop]
	professions?: [...#Profession]
	conditions?: [...#Condition]
	bodyParts?: [...#BodyPart]
	skills?: [...#Skill]
	skillCategories?: [...#SkillCategory]
	attackTypes?: [...#AttackType]
	interiorRefs?: [...#InteriorRef]
	interiorProfile?: #InteriorProfile
	weathers?: [...#Weather]
	shelterEffects?: [...#ShelterEffect]
	waterEffects?: [...#WaterEffect]
	foliageEffects?: [...#FoliageEffect]
	hungerStages?: [...#HungerStage]
	fatigueStages?: [...#FatigueStage]
	freshnessStages?: [...#FreshnessStage]
	materials?: [...#MaterialDef]
	toolCategories?: [...#ToolCategoryDef]
	timesOfDay?: [...#TimeOfDay]
	seasons?: [...#Season]
	dungeons?: [...#Dungeon]
	settlements?: [...#Settlement]
	dialogues?: [...#Dialogue]

	// id は表ごとに一意にする。重複すると同じ id に別の行番号が入り、値が食い違う
	facilityIds_unique: {if facilities != _|_ for i, x in facilities {(x.id): i}} @rawgen(-)
	landmarkIds_unique: {if landmarks != _|_ for i, x in landmarks {(x.id): i}} @rawgen(-)
	scatterZoneIds_unique: {if scatterZones != _|_ for i, x in scatterZones {(x.id): i}} @rawgen(-)
	mapGlyphIds_unique: {if mapGlyphs != _|_ for i, x in mapGlyphs {(x.id): i}} @rawgen(-)
	settlementIds_unique: {if settlements != _|_ for i, x in settlements {(x.id): i}} @rawgen(-)
	bodyPartIds_unique: {if bodyParts != _|_ for i, x in bodyParts {(x.id): i}} @rawgen(-)
	conditionIds_unique: {if conditions != _|_ for i, x in conditions {(x.id): i}} @rawgen(-)
	attackTypeIds_unique: {if attackTypes != _|_ for i, x in attackTypes {(x.id): i}} @rawgen(-)
	skillIds_unique: {if skills != _|_ for i, x in skills {(x.id): i}} @rawgen(-)
	skillCategoryIds_unique: {if skillCategories != _|_ for i, x in skillCategories {(x.id): i}} @rawgen(-)
	interiorRefIds_unique: {if interiorRefs != _|_ for i, x in interiorRefs {(x.id): i}} @rawgen(-)
	dialogueIds_unique: {if dialogues != _|_ for i, x in dialogues {(x.id): i}} @rawgen(-)
	dungeonIds_unique: {if dungeons != _|_ for i, x in dungeons {(x.id): i}} @rawgen(-)
	interiorContentIds_unique: {if interiorContents != _|_ for i, x in interiorContents {(x.id): i}} @rawgen(-)

	// アイテムグループとアイテムコレクションは同じ id の空間で参照される
	itemGroupIds_unique: {
		if itemGroups != _|_ for i, x in itemGroups {(x.id): "itemGroups.\(i)"}
		if itemCollections != _|_ for i, x in itemCollections {(x.id): "itemCollections.\(i)"}
	} @rawgen(-)

	// タイルの生成の仕様は同じ表のタイルを指す。#TileSpawn の中で索引を引くと、索引が tiles 自身の評価を待って
	// 循環し、実在するタイルも見つからない。表の外で引く
	tileSpawns_ref: {
		if tiles != _|_ for x in tiles if x.spawn.tile != _|_ {(x.id): _tileIDs[x.spawn.tile]}
	} @rawgen(-)

	// 熟練の効果倍率の id はスキルをまたいで一意にする
	proficiencyIds_unique: {
		if skills != _|_ for i, s in skills if s.proficiencies != _|_ for j, x in s.proficiencies {(x.id): "skills.\(i).proficiencies.\(j)"}
	} @rawgen(-)
}

// 読了に必要な総読書量
#ReadingEffort: int & >=1 & <=9999

// レシピ
#Recipe: {
	id!:    #EntityID
	id_ref: _itemIDs[id] @rawgen(-)
	name!:  #EntityName
	inputs!: [...#RecipeInput]
}

// レシピ素材
#RecipeInput: {
	id!:     #EntityID
	id_ref:  _itemIDs[id] @rawgen(-)
	amount!: #MaterialAmount
}

// 不調が未治療のときの振る舞いと治し方
#RecoveryMode: "recover_after_tend" | "progress_until_tend" | "recover_over_time"

// リロードに必要な行動力
#ReloadEffort: int & >=1 & <=100

// 治療する性質。Treats に一致する不調のうち最も重い1つを治療済みにする
#Remedy: {
	treats!: [...#ConditionTypeName]
	treats_ref: [for x in treats {_conditionIDs[x]}] @rawgen(-)
	potency!: #Potency
}

// 奥室の役割名と内装レシピの対
#RoomContent: {
	role!:       #RoomRole
	content!:    #EntityID
	content_ref: _interiorContentIDs[content] @rawgen(-)
}

// 部屋の役割名。interior の roleName に相当する。main/back は Go 内部専用で raw.toml には現れないため含めない
#RoomRole: "bath" | "bedroom" | "coldroom" | "corridor" | "dressing" | "examination_room" | "genkan" | "kitchen" | "living" | "office" | "pharmacy" | "restroom" | "storage" | "storeroom" | "toilet" | "waiting"

// 散布 prop 1種。ref の空文字は置かないことを表す
#ScatterEntry: {
	ref!: string
	if ref != "" {
		ref_ref: _propIDs[ref] @rawgen(-)
	}
	weight!: int32 & >=1
	big!:    bool
	satellites?: [...#PropSpot]
	if satellites != _|_ {
		satellites_ref: [for x in satellites {_propIDs[x.name]}] @rawgen(-)
	}
}

// 開けた地形の散布ゾーン1種。grass は草を置くタイルに、soil は草の生えていない土へ soilDensity の確率で置く地面の飾り
#ScatterZone: {
	id!:           #EntityID
	grassDensity!: number
	soilDensity!:  number
	propDensity!:  number
	lootGroup!:    #EntityID
	lootGroup_ref: _itemGroupIDs[lootGroup] @rawgen(-)
	entries!: [...#ScatterEntry]
	grass!: [...#GroundCoverEntry]
	grass_ref: [for x in grass {_propIDs[x.ref]}] @rawgen(-)
	soil!: [...#GroundCoverEntry]
	soil_ref: [for x in soil {_propIDs[x.ref]}] @rawgen(-)
}

// 季節の表示名と世界温度。春と秋は年の前半と後半の中点、夏は頂、冬は底の温度で、間を正弦波でつなぐ
#Season: {
	id!:   #SeasonId
	name!: #EntityName
	temp!: #SeasonTemp
}

// 季節
#SeasonId: "spring" | "summer" | "autumn" | "winter"

// 季節の世界温度。℃
#SeasonTemp: int & >=-100 & <=100

// 感覚。命中率と回避率に影響する
#Sensation: int & >=-100 & <=100

// 原野の小集落1種。当選したチャンクで weight の比で種類を選び、中心から相対の位置に住人の member と prop を置く。
// id は地図の記号の id を兼ねる。行の順に重みを積む
#Settlement: {
	id!:     #EntityID
	weight!: int32 & >=1

	// 住人の member。name は members の id
	members!: [...#PropSpot]
	members_ref: [for x in members {_memberIDs[x.name]}] @rawgen(-)
	props!: [...#PropSpot]
	props_ref: [for x in props {_propIDs[x.name]}] @rawgen(-)
}

// タイルの囲われの度合い。屋内外の判定と、世界温度の受け方に効く
#Shelter: "outdoor" | "partial" | "indoor"

// 囲われの度合いごとの、世界温度の受け方と表示名。受ける温度は anchorTemp + (世界温度 - anchorTemp) * influencePercent / 100
#ShelterEffect: {
	id!:               #Shelter
	name!:             #EntityName
	anchorTemp!:       #TileTempModifier
	influencePercent!: #ShelterInfluencePercent

	// 屋根か壁に覆われているか。覆われないタイルに繋がる領域は屋外とみなす
	covered!: bool
}

// 世界温度を引き寄せる割合。百分率。100 なら世界温度をそのまま受ける
#ShelterInfluencePercent: int & >=0 & <=100

// 敷地の類型の、民家と店それぞれで選ばれる重みと前庭の奥行き。行の順に重みを積む
#SiteChoice: {
	id!:         #SiteType
	homeWeight!: #SiteWeight
	shopWeight!: #SiteWeight
	yardDepth!:  #YardDepth

	// 敷地を塀で囲うか。街路に面して開く敷地は囲わない
	fence!: bool
}

// 敷地の類型。前庭の深さと外構の出し分けに使う
#SiteType: "detached" | "shopfront" | "roadside"

// 敷地の類型の選ばれる重み。0 ならその種類の建物では選ばれない
#SiteWeight: int32 & >=0 @rawgen(type=int32)

// スキル。同じカテゴリのスキルは行の順に表示する
#Skill: {
	id!:          #SkillId
	name!:        #EntityName
	summary!:     #EntityDescription
	gainedBy!:    #EntityDescription
	effect!:      #EntityDescription
	ability!:     #AbilityId
	category!:    #SkillCategoryId
	category_ref: _skillCategoryIDs[category] @rawgen(-)
	proficiencies?: [...#SkillProficiency]
}

// スキル本設定
#SkillBook: {
	targetSkill!:    #SkillId
	targetSkill_ref: _skillIDs[targetSkill] @rawgen(-)
	requiredLevel!:  #SkillLevel
	maxLevel!:       #SkillLevel
}

// スキルのカテゴリ。行の順に表示する
#SkillCategory: {
	id!:   #SkillCategoryId
	name!: #EntityName
}

// スキルのカテゴリの id
#SkillCategoryId: string & strings.MinRunes(
			1) & strings.MaxRunes(
			50) & =~"^[a-z][a-z0-9_]*$"

// スキルID
#SkillId: "sword" | "spear" | "fist" | "weight_bearing" | "bow" | "handgun" | "rifle" | "cannon" | "exploration" | "crafting" | "smithing" | "negotiation" | "mechanic" | "sprinting" | "stealth" | "night_vision" | "cold_resist" | "hunger_resist" | "healing" | "heavy_armor" | "fire_resist" | "thunder_resist" | "chill_resist" | "photon_resist" @rawgen(prefix)

// スキルレベル
#SkillLevel: int & >=0 & <=100

// スキルが左右する熟練の効果倍率
#SkillProficiency: {
	id!:       #ProficiencyId
	perLevel!: #ProficiencyPerLevel

	// この倍率が受けるダメージを軽減する攻撃属性。無属性を除く属性それぞれをちょうど1つの倍率が持つ
	resists?: #Element
}

// 天候のスペルの基準長。ターン
#SpellTurns: int & >=1 & <=100000

// スプライト描画深度
#SpriteDepth: 0 | 1 | 2 | 3 | 4 @rawgen(enum=float32)

// スプライトキー
#SpriteKey: string & strings.MinRunes(
		1) & strings.MaxRunes(
		100) & =~"^[a-z][a-z0-9_]*$"

// スプライトレンダー設定
#SpriteRender: {
	spriteSheetName!: #SpriteSheetName
	spriteKey!:       #SpriteKey
	depth!:           #SpriteDepth
}

// スプライトシート
#SpriteSheet: {
	name!: #EntityName
	path!: #ImagePath
}

// スプライトシート名
#SpriteSheetName: string & strings.MinRunes(
			1) & strings.MaxRunes(
			50) & =~"^[a-z][a-z0-9_]*$"

// ステージの基本気温。℃
#StageBaseTemp: int & >=-100 & <=100

// 1段階の長さ。この経過ターンごとに新鮮→劣化→腐敗と進む。省略すると腐敗しない
#StageLengthTurns: int & >=1 & <=100000

// 収納ローデータ
#StorageRaw: {
	// 収納の最大格納重量
	maxWeight!: #Weight

	// 初期アイテムの抽選に使うItemTable名
	lootTableId?: #EntityID

	// 初期アイテム数のダイス表記。省略時は1
	lootCount?: #Dice
}

// 筋力。物理ダメージに影響する
#Strength: int & >=-100 & <=100

// 配置指示の種別。家具・戦利品・敵・装飾・罠を同じ器で扱う
#StuffKind: "furniture" | "loot" | "being" | "decor" | "trap"

// ターゲットグループ
#TargetGroup: "ENEMY" | "ALLY" | "WEAPON" | "NONE" @rawgen(prefix)

// ターゲット数
#TargetNum: "SINGLE" | "ALL"

// タイル
#Tile: {
	id!:           #EntityID
	name!:         #EntityName
	description!:  #EntityDescription
	blockPass!:    #BlocksPassage
	blockView!:    #BlocksView
	spriteRender!: #SpriteRender

	// 水のタイル。上に立つと水中、隣に立つと水辺の気温の補正を受ける
	water?: bool

	// 植生。省くと植生なし
	foliage?: #Foliage

	// マップの計画からこのタイルを生成するときの仕様。省くと計画に置けない
	spawn?: #TileSpawn
}

// 計画のタイルを実体化する仕様
#TileSpawn: {
	// 生成するタイルの id。壁の計画を壁の見た目の別タイルで生成するときに使う。省くと自身
	tile?: #EntityID

	// 周りのタイルを見てオートタイルの添字を計算するか。単一の絵柄のタイルは false
	autotile!: bool
}

// 水や植生がタイルの気温に足す補正。℃
#TileTempModifier: int & >=-50 & <=50

// 時間帯の表示名、世界温度への加算、地上の日照の明るさと環境光の色。
// 明るさと色は時間帯の中心の値で、間は隣の時間帯の値と線形に補間する
#TimeOfDay: {
	id!:           #TimeOfDayId
	name!:         #EntityName
	tempModifier!: #TimeOfDayTempModifier
	daylight!:     #LightLevel
	ambientColor!: #AmbientColor
}

// 時間帯。1日は夜明けで始まり、この順に巡る
#TimeOfDayId: "dawn" | "morning" | "noon" | "evening" | "night" | "midnight"

// 時間帯による世界温度への加算。℃
#TimeOfDayTempModifier: int & >=-50 & <=50

// 道具の性質。道具の分類とグレードを持つ
#Tool: {
	categories!: [...#ToolCategory]
	grade!: #ToolGrade
}

// 道具の分類
#ToolCategory: "prying" | "precision" | "cutting" | "chopping" | "mining"

// 道具の分類と、合う道具で壊したとき伸びるスキル。スキルを省くと伸びない
#ToolCategoryDef: {
	id!:    #ToolCategory
	skill?: #SkillId
}

// 道具のグレード。高いほど速く壊し多く得る
#ToolGrade: int & >=1 & <=3

// 使用可能シーン
#UsableScene: "BATTLE" | "FIELD" | "ANY"

// 建物の変種の重み
#VariantChoice: {
	id!:     #BuildingVariant
	weight!: #EntryWeight

	// 引いた損傷段階を上書きする。省くと上書きしない
	damage?: #BuildingDamage

	// 散らかり段階をこれ以上に引き上げる。省くと引き上げない
	minClutter?: #BuildingClutter

	// 引いた家具量の倍率を上書きする。省くと上書きしない
	density?: #DensityFactor
}

// AI視界距離（タイル単位）
#ViewDistance: int & >=1 & <=100

// 体力。HPに影響する
#Vitality: int & >=-100 & <=100

// 次階層ワープトリガー
#WarpNextTriggerRaw: {}

// 前階層ワープトリガー
#WarpPrevTriggerRaw: {}

// 水がタイルの気温に足す補正
#WaterEffect: {
	id!:           #WaterExposure
	tempModifier!: #TileTempModifier
}

// タイルが受ける水の影響
#WaterExposure: "nearby" | "submerged"

// 装備可能設定
#Wearable: {
	defense!:           #ArmorDefense
	equipmentCategory!: #EquipmentCategory
	insulationCold!:    #InsulationCold
	insulationHeat!:    #InsulationHeat
}

// 天候。遷移の粘りの距離は行の順でなく Go の WeatherKind の並びに依る
#Weather: {
	id!:            #WeatherId
	name!:          #EntityName
	tempModifier!:  #WeatherTempModifier
	visionDelta!:   #WeatherVisionDelta
	severity!:      #WeatherSeverity
	spellTurnsMin!: #SpellTurns
	spellTurnsMax!: #SpellTurns
	seasons!: [...#WeatherSeasonWeight]
}

// 天候の種類
#WeatherId: "clear" | "cloudy" | "rain" | "snow" | "blizzard" | "cold_snap"

// 季節ごとの天候の出やすさ
#WeatherSeasonWeight: {
	id!:     #SeasonId
	weight!: #EntryWeight
}

// 天候の寒さの厳しさ。奥地ほど厳しい天候を厚くし、冬と奥地で長引かせるのに使う
#WeatherSeverity: int & >=0 & <=10

// 天候による世界温度への加算。℃
#WeatherTempModifier: int & >=-50 & <=50

// 天候による視程レンジへの加算。タイル。負で狭くなる
#WeatherVisionDelta: int & >=-100 & <=100

// 重量。単位付き文字列で指定する。例: "500 g" "2 kg" "1 mg"
#Weight: string & =~"^[0-9]+(\\.[0-9]+)? (mg|g|kg)$"

// 入口側に空ける前庭の奥行き。タイル
#YardDepth: int32 & >=0 & <=10 @rawgen(type=int32)

// 市街地の地区
#Zone: "downtown" | "residential" | "industrial"
