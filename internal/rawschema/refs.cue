package rawschema

// 行の間の参照を引く id の索引。参照する側の定義は <フィールド>_ref で索引を引き、無い id は undefined field になる。
// 索引は隠しフィールドに置き、Go の型にもデータにも出さない。

// 索引が表を名前で引けるよう、#Raws の表をパッケージの最上位にも置く。省いた表は索引を空にする
items?:            _
bodyAttacks?:      _
members?:          _
commandTables?:    _
dropTables?:       _
itemGroups?:       _
itemCollections?:  _
itemTables?:       _
enemyTables?:      _
facilities?:       _
tiles?:            _
props?:            _
remains?:          _
decors?:           _
conditions?:       _
bodyParts?:        _
skills?:           _
skillCategories?:  _
dialogues?:        _
layouts?:          _
rectRoomStarts?:   _
noiseStarts?:      _
buildingStarts?:   _
clearingStarts?:   _
templateStarts?:   _
simpleStarts?:     _
corridorSteps?:    _
doorSteps?:        _
bigRoomSteps?:     _
automataSteps?:    _
widenSteps?:       _
scatterWallSteps?: _
ruinedWallSteps?:  _
ruinedPathSteps?:  _
treeSteps?:        _
trailSteps?:       _
gladeSteps?:       _
simpleSteps?:      _

_itemIDs: {if items != _|_ for x in items {(x.id): true}}
_bodyAttackIDs: {if bodyAttacks != _|_ for x in bodyAttacks {(x.id): true}}
_meleeItemIDs: {if items != _|_ for x in items if x.melee != _|_ {(x.id): true}}
_memberIDs: {if members != _|_ for x in members {(x.id): true}}
_commandTableIDs: {if commandTables != _|_ for x in commandTables {(x.id): true}}
_dropTableIDs: {if dropTables != _|_ for x in dropTables {(x.id): true}}
_itemTableIDs: {if itemTables != _|_ for x in itemTables {(x.id): true}}
_enemyTableIDs: {if enemyTables != _|_ for x in enemyTables {(x.id): true}}
_facilityIDs: {if facilities != _|_ for x in facilities {(x.id): true}}
_tileIDs: {if tiles != _|_ for x in tiles {(x.id): true}}
_propIDs: {if props != _|_ for x in props {(x.id): true}}
_remainsIDs: {if remains != _|_ for x in remains {(x.id): true}}
_decorIDs: {if decors != _|_ for x in decors {(x.id): true}}
_conditionIDs: {if conditions != _|_ for x in conditions {(x.id): true}}
_bodyPartIDs: {if bodyParts != _|_ for x in bodyParts {(x.id): true}}
_skillIDs: {if skills != _|_ for x in skills {(x.id): true}}
_skillCategoryIDs: {if skillCategories != _|_ for x in skillCategories {(x.id): true}}
_dialogueIDs: {if dialogues != _|_ for x in dialogues {(x.id): true}}

_layoutIDs: {if layouts != _|_ for x in layouts {(x.id): true}}

// タイルに ground を書き、ダンジョンの行の下地を使う間取り
_groundLayoutIDs: {
	if layouts != _|_ for x in layouts if x.tiles.floor == "ground" || x.tiles.wall == "ground" || x.tiles.void == "ground" {(x.id): true}
}

// 最初の段の表はどれも同じ id の空間で参照される
_startIDs: {
	if rectRoomStarts != _|_ for x in rectRoomStarts {(x.id): true}
	if noiseStarts != _|_ for x in noiseStarts {(x.id): true}
	if buildingStarts != _|_ for x in buildingStarts {(x.id): true}
	if clearingStarts != _|_ for x in clearingStarts {(x.id): true}
	if templateStarts != _|_ for x in templateStarts {(x.id): true}
	if simpleStarts != _|_ for x in simpleStarts {(x.id): true}
}

// 中の段の表はどれも同じ id の空間で参照される
_stepIDs: {
	if corridorSteps != _|_ for x in corridorSteps {(x.id): true}
	if doorSteps != _|_ for x in doorSteps {(x.id): true}
	if bigRoomSteps != _|_ for x in bigRoomSteps {(x.id): true}
	if automataSteps != _|_ for x in automataSteps {(x.id): true}
	if widenSteps != _|_ for x in widenSteps {(x.id): true}
	if scatterWallSteps != _|_ for x in scatterWallSteps {(x.id): true}
	if ruinedWallSteps != _|_ for x in ruinedWallSteps {(x.id): true}
	if ruinedPathSteps != _|_ for x in ruinedPathSteps {(x.id): true}
	if treeSteps != _|_ for x in treeSteps {(x.id): true}
	if trailSteps != _|_ for x in trailSteps {(x.id): true}
	if gladeSteps != _|_ for x in gladeSteps {(x.id): true}
	if simpleSteps != _|_ for x in simpleSteps {(x.id): true}
}

// アイテムグループとアイテムコレクションは同じ id の空間で参照される
_itemGroupIDs: {
	if itemGroups != _|_ for x in itemGroups {(x.id): true}
	if itemCollections != _|_ for x in itemCollections {(x.id): true}
}
