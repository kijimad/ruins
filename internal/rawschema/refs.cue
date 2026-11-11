package rawschema

// 行の間の参照を引く id の索引。参照する側の定義は <フィールド>_ref で索引を引き、無い id は undefined field になる。
// 索引は隠しフィールドに置き、Go の型にもデータにも出さない。

// 索引が表を名前で引けるよう、#Raws の表をパッケージの最上位にも置く。省いた表は索引を空にする
items?:            _
members?:          _
commandTables?:    _
dropTables?:       _
itemGroups?:       _
itemCollections?:  _
itemTables?:       _
enemyTables?:      _
facilities?:       _
interiorContents?: _
tiles?:            _
props?:            _
conditions?:       _
bodyParts?:        _
skills?:           _
skillCategories?:  _
interiorRefs?:     _
dialogues?:        _

_itemIDs: {if items != _|_ for x in items {(x.id): true}}
_memberIDs: {if members != _|_ for x in members {(x.id): true}}
_commandTableIDs: {if commandTables != _|_ for x in commandTables {(x.id): true}}
_dropTableIDs: {if dropTables != _|_ for x in dropTables {(x.id): true}}
_itemTableIDs: {if itemTables != _|_ for x in itemTables {(x.id): true}}
_enemyTableIDs: {if enemyTables != _|_ for x in enemyTables {(x.id): true}}
_facilityIDs: {if facilities != _|_ for x in facilities {(x.id): true}}
_interiorContentIDs: {if interiorContents != _|_ for x in interiorContents {(x.id): true}}
_tileIDs: {if tiles != _|_ for x in tiles {(x.id): true}}
_propIDs: {if props != _|_ for x in props {(x.id): true}}
_conditionIDs: {if conditions != _|_ for x in conditions {(x.id): true}}
_bodyPartIDs: {if bodyParts != _|_ for x in bodyParts {(x.id): true}}
_skillIDs: {if skills != _|_ for x in skills {(x.id): true}}
_skillCategoryIDs: {if skillCategories != _|_ for x in skillCategories {(x.id): true}}
_dialogueIDs: {if dialogues != _|_ for x in dialogues {(x.id): true}}

// アイテムグループとアイテムコレクションは同じ id の空間で参照される
_itemGroupIDs: {
	if itemGroups != _|_ for x in itemGroups {(x.id): true}
	if itemCollections != _|_ for x in itemCollections {(x.id): true}
}

// prop を持つ内装の Ref。外皮、外構、hero の目玉は prop として置くので、これだけを指せる
_propRefIDs: {if interiorRefs != _|_ for x in interiorRefs if x.prop != _|_ {(x.id): true}}
