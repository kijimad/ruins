package editor

var templateText = `
{{define "common-head"}}
  <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet">
  <link href="https://cdn.jsdelivr.net/npm/tom-select@2.4.3/dist/css/tom-select.bootstrap5.min.css" rel="stylesheet">
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
  <script src="https://cdn.jsdelivr.net/npm/tom-select@2.4.3/dist/js/tom-select.complete.min.js"></script>
{{end}}

{{define "tom-select-init"}}
<script>
function initTomSelect(root) {
  (root || document).querySelectorAll('select.form-select:not(.tomselected)').forEach(function(el) {
    new TomSelect(el, {create: false, allowEmptyOption: true, maxOptions: null});
  });
}
initTomSelect();
document.body.addEventListener('htmx:afterSettle', function(e) {
  initTomSelect(e.detail.target);
});
</script>
{{end}}

{{define "header"}}
<nav class="navbar navbar-dark bg-dark border-bottom px-3" style="height:40px;min-height:40px;" data-bs-theme="dark">
  <a class="navbar-brand py-0" href="/" style="font-size:14px;">Ruins Editor</a>
</nav>
{{end}}

{{define "dashboard"}}
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Ruins Editor</title>
  {{template "common-head"}}
  <style>
    .dash-card { text-decoration:none; color:inherit; }
    .dash-card:hover .card { border-color:#0d6efd; box-shadow:0 0 0 1px #0d6efd; }
    .dash-card .card { transition: border-color 0.15s, box-shadow 0.15s; }
  </style>
</head>
<body>
  {{template "header" .}}
  <div class="container py-4" style="max-width:900px;">
    <h4 class="mb-4">Ruins Editor</h4>

    <h6 class="text-secondary mb-2">エンティティ</h6>
    <div class="row g-3 mb-4">
      <div class="col-md-3"><a class="dash-card" href="/items"><div class="card"><div class="card-body py-2 px-3"><strong>Items</strong><div class="text-muted" style="font-size:12px;">アイテム定義</div></div></div></a></div>
      <div class="col-md-3"><a class="dash-card" href="/members"><div class="card"><div class="card-body py-2 px-3"><strong>Members</strong><div class="text-muted" style="font-size:12px;">メンバー定義</div></div></div></a></div>
      <div class="col-md-3"><a class="dash-card" href="/tiles"><div class="card"><div class="card-body py-2 px-3"><strong>Tiles</strong><div class="text-muted" style="font-size:12px;">タイル定義</div></div></div></a></div>
      <div class="col-md-3"><a class="dash-card" href="/props"><div class="card"><div class="card-body py-2 px-3"><strong>Props</strong><div class="text-muted" style="font-size:12px;">配置物定義</div></div></div></a></div>
    </div>

    <h6 class="text-secondary mb-2">テーブル</h6>
    <div class="row g-3 mb-4">
      <div class="col-md-3"><a class="dash-card" href="/recipes"><div class="card"><div class="card-body py-2 px-3"><strong>Recipes</strong><div class="text-muted" style="font-size:12px;">レシピ</div></div></div></a></div>
      <div class="col-md-3"><a class="dash-card" href="/command-tables"><div class="card"><div class="card-body py-2 px-3"><strong>CmdTbl</strong><div class="text-muted" style="font-size:12px;">コマンドテーブル</div></div></div></a></div>
      <div class="col-md-3"><a class="dash-card" href="/drop-tables"><div class="card"><div class="card-body py-2 px-3"><strong>DropTbl</strong><div class="text-muted" style="font-size:12px;">ドロップテーブル</div></div></div></a></div>
      <div class="col-md-3"><a class="dash-card" href="/item-tables"><div class="card"><div class="card-body py-2 px-3"><strong>ItemTbl</strong><div class="text-muted" style="font-size:12px;">アイテムテーブル</div></div></div></a></div>
      <div class="col-md-3"><a class="dash-card" href="/enemy-tables"><div class="card"><div class="card-body py-2 px-3"><strong>EnemyTbl</strong><div class="text-muted" style="font-size:12px;">敵テーブル</div></div></div></a></div>
      <div class="col-md-3"><a class="dash-card" href="/professions"><div class="card"><div class="card-body py-2 px-3"><strong>Professions</strong><div class="text-muted" style="font-size:12px;">職業定義</div></div></div></a></div>
    </div>

    <h6 class="text-secondary mb-2">マップ</h6>
    <div class="row g-3 mb-4">
      <div class="col-md-3"><a class="dash-card" href="/palettes"><div class="card"><div class="card-body py-2 px-3"><strong>Palettes</strong><div class="text-muted" style="font-size:12px;">パレット定義</div></div></div></a></div>
      <div class="col-md-3"><a class="dash-card" href="/layouts"><div class="card"><div class="card-body py-2 px-3"><strong>Layouts</strong><div class="text-muted" style="font-size:12px;">レイアウト編集</div></div></div></a></div>
    </div>

    <h6 class="text-secondary mb-2">スプライト</h6>
    <div class="row g-3 mb-4">
      <div class="col-md-3"><a class="dash-card" href="/sprite-sheets"><div class="card"><div class="card-body py-2 px-3"><strong>Sheets</strong><div class="text-muted" style="font-size:12px;">スプライトシート</div></div></div></a></div>
      <div class="col-md-3"><a class="dash-card" href="/cutter"><div class="card"><div class="card-body py-2 px-3"><strong>Cutter</strong><div class="text-muted" style="font-size:12px;">スプライト切り出し</div></div></div></a></div>
    </div>
  </div>
</body>
</html>
{{end}}

{{define "index"}}
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Ruins Editor - Items</title>
  {{template "common-head"}}
  <style>
    .sidebar { width: 280px; min-width: 280px; overflow-y: auto; }
    .item-entry { cursor: pointer; padding: 4px 8px; border-bottom: 1px solid #dee2e6; display: flex; align-items: center; gap: 6px; font-size: 13px; }
    .item-entry:hover { background: rgba(0,0,0,0.04); }
    .item-entry.active { background: rgba(13,110,253,0.25); border-left: 3px solid #0d6efd; }
    .main-content { flex: 1; overflow-y: auto; padding: 24px; }
    .content-area { height: calc(100vh - 40px); }
  </style>
</head>
<body style="overflow:hidden;">
  {{template "header" .}}
  <div class="d-flex content-area">
    <div class="sidebar border-end p-0 d-flex flex-column">
      <div class="p-2 border-bottom">
        <form hx-post="/items/new" hx-target="#edit-panel" hx-swap="innerHTML" class="d-flex gap-1">
          <input type="text" class="form-control form-control-sm" name="name" required placeholder="新規アイテム">
          <button type="submit" class="btn btn-primary btn-sm">追加</button>
        </form>
      </div>
      <div id="item-count" class="p-1 border-bottom text-secondary" style="font-size:12px;">
        {{len .Items}} items
      </div>
      <div id="item-list" style="overflow-y:auto;flex:1;">
        {{range .Items}}
        {{template "item-entry" .}}
        {{end}}
      </div>
    </div>
    <div class="main-content" id="edit-panel">
      {{if .Edit}}
      {{template "item-edit" .Edit}}
      {{else}}
      <div class="text-secondary mt-5 text-center">アイテムを選択してください</div>
      {{end}}
    </div>
  </div>
  {{template "tom-select-init"}}
</body>
</html>
{{end}}

{{define "item-entry"}}
<div class="item-entry{{if .Active}} active{{end}}" id="entry-{{.Index}}"
     hx-get="/items/{{.Index}}/edit" hx-target="#edit-panel" hx-swap="innerHTML"
     onclick="document.querySelectorAll('.item-entry').forEach(e=>e.classList.remove('active'));this.classList.add('active');">
  <span style="{{spriteStyle .Item.SpriteSheetName .Item.SpriteKey 1}}" class="flex-shrink-0"></span>
  <span class="text-truncate flex-grow-1">{{.Item.Name}}</span>
  <span class="flex-shrink-0">
    {{- if isNotNil .Item.Melee}}<span class="badge text-bg-danger">近</span>{{end -}}
    {{- if isNotNil .Item.Fire}}<span class="badge text-bg-danger">射</span>{{end -}}
    {{- if isNotNil .Item.Wearable}}<span class="badge text-bg-info">防</span>{{end -}}
    {{- if isNotNil .Item.Consumable}}<span class="badge text-bg-success">消</span>{{end -}}
    {{- if isNotNil .Item.Ammo}}<span class="badge text-bg-warning">弾</span>{{end -}}
    {{- if isNotNil .Item.Book}}<span class="badge text-bg-secondary">本</span>{{end -}}
  </span>
</div>
{{end}}

{{define "item-list-oob"}}
<div id="item-list" hx-swap-oob="innerHTML:#item-list">
{{range .Items}}
{{template "item-entry" .}}
{{end}}
</div>
{{end}}

{{define "item-count-oob"}}
<div id="item-count" hx-swap-oob="innerHTML:#item-count">
  {{len .Items}} items
</div>
{{end}}

{{define "select-target-group"}}
<select class="form-select" name="{{.Name}}">
  <option value="ALLY" {{selected .Value "ALLY"}}>ALLY (味方)</option>
  <option value="ENEMY" {{selected .Value "ENEMY"}}>ENEMY (敵)</option>
  <option value="WEAPON" {{selected .Value "WEAPON"}}>WEAPON (武器)</option>
  <option value="NONE" {{selected .Value "NONE"}}>NONE (なし)</option>
</select>
{{end}}

{{define "select-target-num"}}
<select class="form-select" name="{{.Name}}">
  <option value="SINGLE" {{selected .Value "SINGLE"}}>SINGLE (単体)</option>
  <option value="ALL" {{selected .Value "ALL"}}>ALL (全体)</option>
</select>
{{end}}

{{define "select-usable-scene"}}
<select class="form-select" name="{{.Name}}">
  <option value="ANY" {{selected .Value "ANY"}}>ANY (いつでも)</option>
  <option value="BATTLE" {{selected .Value "BATTLE"}}>BATTLE (戦闘)</option>
  <option value="FIELD" {{selected .Value "FIELD"}}>FIELD (フィールド)</option>
</select>
{{end}}

{{define "select-element"}}
<select class="form-select" name="{{.Name}}">
  <option value="NONE" {{selected .Value "NONE"}}>NONE (無)</option>
  <option value="FIRE" {{selected .Value "FIRE"}}>FIRE (火)</option>
  <option value="THUNDER" {{selected .Value "THUNDER"}}>THUNDER (雷)</option>
  <option value="CHILL" {{selected .Value "CHILL"}}>CHILL (氷)</option>
  <option value="PHOTON" {{selected .Value "PHOTON"}}>PHOTON (光)</option>
</select>
{{end}}

{{define "select-attack-category"}}
<select class="form-select" name="{{.Name}}">
  <option value="SWORD" {{selected .Value "SWORD"}}>SWORD (刀剣)</option>
  <option value="SPEAR" {{selected .Value "SPEAR"}}>SPEAR (長物)</option>
  <option value="FIST" {{selected .Value "FIST"}}>FIST (格闘)</option>
  <option value="HANDGUN" {{selected .Value "HANDGUN"}}>HANDGUN (拳銃)</option>
  <option value="RIFLE" {{selected .Value "RIFLE"}}>RIFLE (小銃)</option>
  <option value="CANON" {{selected .Value "CANON"}}>CANON (大砲)</option>
  <option value="BOW" {{selected .Value "BOW"}}>BOW (弓)</option>
</select>
{{end}}

{{define "select-equipment-category"}}
<select class="form-select" name="{{.Name}}">
  <option value="HEAD" {{selected .Value "HEAD"}}>HEAD (頭部)</option>
  <option value="TORSO" {{selected .Value "TORSO"}}>TORSO (胴体)</option>
  <option value="ARMS" {{selected .Value "ARMS"}}>ARMS (腕部)</option>
  <option value="HANDS" {{selected .Value "HANDS"}}>HANDS (手部)</option>
  <option value="LEGS" {{selected .Value "LEGS"}}>LEGS (脚部)</option>
  <option value="FEET" {{selected .Value "FEET"}}>FEET (足部)</option>
  <option value="JEWELRY" {{selected .Value "JEWELRY"}}>JEWELRY (装飾)</option>
</select>
{{end}}

{{define "sprite-grid"}}
{{$sheet := .SheetName}}
{{range .Keys}}
<div class="sprite-option border rounded p-1 text-center" style="cursor:pointer;width:36px;" data-key="{{.}}" onclick="pickSprite(this)" title="{{.}}">
  <span style="{{spriteStyle $sheet . 1}}"></span>
</div>
{{end}}
{{end}}

{{define "members"}}
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Ruins Editor - Members</title>
  {{template "common-head"}}
  <style>
    .sidebar { width: 280px; min-width: 280px; overflow-y: auto; }
    .member-entry { cursor: pointer; padding: 4px 8px; border-bottom: 1px solid #dee2e6; display: flex; align-items: center; gap: 6px; font-size: 13px; }
    .member-entry:hover { background: rgba(0,0,0,0.04); }
    .member-entry.active { background: rgba(13,110,253,0.25); border-left: 3px solid #0d6efd; }
    .main-content { flex: 1; overflow-y: auto; padding: 24px; }
    .content-area { height: calc(100vh - 40px); }
  </style>
</head>
<body style="overflow:hidden;">
  {{template "header" .}}
  <div class="d-flex content-area">
    <div class="sidebar border-end p-0 d-flex flex-column">
      <div class="p-2 border-bottom">
        <form hx-post="/members/new" hx-target="#member-edit-panel" hx-swap="innerHTML" class="d-flex gap-1">
          <input type="text" class="form-control form-control-sm" name="name" required placeholder="新規メンバー">
          <button type="submit" class="btn btn-primary btn-sm">追加</button>
        </form>
      </div>
      <div id="member-count" class="p-1 border-bottom text-secondary" style="font-size:12px;">
        {{len .Items}} members
      </div>
      <div id="member-list" style="overflow-y:auto;flex:1;">
        {{range .Items}}
        {{template "member-entry" .}}
        {{end}}
      </div>
    </div>
    <div class="main-content" id="member-edit-panel">
      {{if .Edit}}
      {{template "member-edit" .Edit}}
      {{else}}
      <div class="text-secondary mt-5 text-center">メンバーを選択してください</div>
      {{end}}
    </div>
  </div>
  {{template "tom-select-init"}}
</body>
</html>
{{end}}

{{define "member-entry"}}
<div class="member-entry{{if .Active}} active{{end}}" id="mentry-{{.Index}}"
     hx-get="/members/{{.Index}}/edit" hx-target="#member-edit-panel" hx-swap="innerHTML"
     onclick="document.querySelectorAll('.member-entry').forEach(e=>e.classList.remove('active'));this.classList.add('active');">
  <span style="{{spriteStyle .Member.SpriteSheetName .Member.SpriteKey 1}}" class="flex-shrink-0"></span>
  <span class="text-truncate flex-grow-1">{{.Member.Name}}</span>
  <span class="flex-shrink-0">
    {{- if derefBool .Member.Player}}<span class="badge text-bg-primary">PC</span>{{end -}}
    {{- if .Member.IsBoss}}<span class="badge text-bg-danger">Boss</span>{{end -}}
    {{- if eq .Member.FactionType "FactionAlly"}}<span class="badge text-bg-success">味方</span>{{end -}}
    {{- if eq .Member.FactionType "FactionEnemy"}}<span class="badge text-bg-warning">敵</span>{{end -}}
    {{- if eq .Member.FactionType "FactionNeutral"}}<span class="badge text-bg-secondary">中立</span>{{end -}}
  </span>
</div>
{{end}}

{{define "member-list-oob"}}
<div id="member-list" hx-swap-oob="innerHTML:#member-list">
{{range .Items}}
{{template "member-entry" .}}
{{end}}
</div>
{{end}}

{{define "member-count-oob"}}
<div id="member-count" hx-swap-oob="innerHTML:#member-count">
  {{len .Items}} members
</div>
{{end}}

{{define "member-edit"}}
{{template "sprite-picker-js"}}
<form hx-post="/members/{{.Index}}" hx-target="#member-edit-panel" hx-swap="innerHTML">
  <div class="d-flex align-items-center gap-3 mb-3">
    <span id="sprite-preview" style="{{spriteStyle .Member.SpriteSheetName .Member.SpriteKey 2}}"></span>
    <h5 class="mb-0 me-auto">{{.Member.Name}}</h5>
    <button class="btn btn-outline-danger btn-sm" type="button" hx-delete="/members/{{.Index}}" hx-target="#member-edit-panel" hx-swap="innerHTML" hx-confirm="削除しますか?">削除</button>
  </div>

  <div class="row g-3 mb-3">
    <div class="col-md-3">
      <label class="form-label">名前</label>
      <input type="text" class="form-control" name="name" value="{{.Member.Name}}" required>
    </div>
    <div class="col-md-3">
      <label class="form-label">スプライトシート</label>
      <select class="form-select" name="sprite_sheet_name"
              onchange="document.getElementById('sprite-key-grid').innerHTML='';document.getElementById('sprite-key-grid').removeAttribute('data-sheet');document.getElementById('sprite-key-input').value='';document.getElementById('sprite-key-display').value='';">
        <option value="">-- 選択 --</option>
        {{range .SheetNames}}
        <option value="{{.}}" {{selected $.Member.SpriteSheetName .}}>{{.}}</option>
        {{end}}
      </select>
    </div>
    <div class="col-md-3 position-relative">
      <label class="form-label">スプライトキー</label>
      <input type="hidden" name="sprite_key" id="sprite-key-input" value="{{.Member.SpriteKey}}">
      <input type="text" class="form-control" id="sprite-key-display" value="{{.Member.SpriteKey}}" readonly onclick="openSpritePicker()" style="cursor:pointer;" placeholder="クリックで選択">
      <div id="sprite-picker-panel" class="d-none position-absolute bg-body border rounded shadow p-2 mt-1" style="z-index:1050;width:400px;right:0;">
        <input type="text" class="form-control form-control-sm mb-2" id="sprite-search" placeholder="検索..." oninput="filterSprites(this.value)">
        <div id="sprite-key-grid" class="d-flex flex-wrap gap-1" style="max-height:200px;overflow-y:auto;"></div>
      </div>
    </div>
    <div class="col-md-3">
      <label class="form-label">AnimKeys</label>
      <input type="text" class="form-control" name="anim_keys" value="{{range $i, $k := .Member.AnimKeys}}{{if $i}}, {{end}}{{$k}}{{end}}" placeholder="key1, key2">
    </div>
  </div>

  <div class="row g-3 mb-3">
    <div class="col-md-3">
      <label class="form-label">所属</label>
      <select class="form-select" name="faction_type">
        <option value="" {{if eq .Member.FactionType ""}}selected{{end}}>-- なし --</option>
        <option value="FactionAlly" {{if eq .Member.FactionType "FactionAlly"}}selected{{end}}>味方</option>
        <option value="FactionEnemy" {{if eq .Member.FactionType "FactionEnemy"}}selected{{end}}>敵</option>
        <option value="FactionNeutral" {{if eq .Member.FactionType "FactionNeutral"}}selected{{end}}>中立</option>
      </select>
    </div>
    <div class="col-md-3">
      <label class="form-label">コマンドテーブル</label>
      <select class="form-select" name="command_table_name">
        <option value="">-- なし --</option>
        {{range .CommandTableNames}}<option value="{{.}}" {{selected $.Member.CommandTableName .}}>{{.}}</option>{{end}}
      </select>
    </div>
    <div class="col-md-3">
      <label class="form-label">ドロップテーブル</label>
      <select class="form-select" name="drop_table_name">
        <option value="">-- なし --</option>
        {{range .DropTableNames}}<option value="{{.}}" {{selected $.Member.DropTableName .}}>{{.}}</option>{{end}}
      </select>
    </div>
  </div>
  <div class="row g-3 mb-3">
    <div class="col-md-3 d-flex align-items-end gap-3">
      <div class="form-check">
        <input class="form-check-input" type="checkbox" name="player" id="player-{{.Index}}" {{if derefBool .Member.Player}}checked{{end}}>
        <label class="form-check-label" for="player-{{.Index}}">プレイヤー</label>
      </div>
      <div class="form-check">
        <input class="form-check-input" type="checkbox" name="is_boss" id="boss-{{.Index}}" {{if .Member.IsBoss}}checked{{end}}>
        <label class="form-check-label" for="boss-{{.Index}}">ボス</label>
      </div>
    </div>
  </div>

  <h6 class="mt-4 mb-2">能力値</h6>
  <div class="row g-3 mb-3">
    <div class="col-md-2">
      <label class="form-label">体力</label>
      <input type="number" class="form-control" name="vitality" value="{{.Member.Abilities.Vitality}}">
    </div>
    <div class="col-md-2">
      <label class="form-label">筋力</label>
      <input type="number" class="form-control" name="strength" value="{{.Member.Abilities.Strength}}">
    </div>
    <div class="col-md-2">
      <label class="form-label">感覚</label>
      <input type="number" class="form-control" name="sensation" value="{{.Member.Abilities.Sensation}}">
    </div>
    <div class="col-md-2">
      <label class="form-label">器用</label>
      <input type="number" class="form-control" name="dexterity" value="{{.Member.Abilities.Dexterity}}">
    </div>
    <div class="col-md-2">
      <label class="form-label">敏捷</label>
      <input type="number" class="form-control" name="agility" value="{{.Member.Abilities.Agility}}">
    </div>
    <div class="col-md-2">
      <label class="form-label">防御</label>
      <input type="number" class="form-control" name="defense" value="{{.Member.Abilities.Defense}}">
    </div>
  </div>

  <h6 class="mt-4 mb-2">
    <input class="form-check-input me-1" type="checkbox" name="has_light" id="light-{{.Index}}" {{if isNotNil .Member.LightSource}}checked{{end}}
           onchange="document.getElementById('light-fields-{{.Index}}').classList.toggle('d-none', !this.checked)">
    <label class="form-check-label" for="light-{{.Index}}">光源</label>
  </h6>
  <div id="light-fields-{{.Index}}" class="row g-3 mb-3 {{if not (isNotNil .Member.LightSource)}}d-none{{end}}">
    <div class="col-md-2">
      <label class="form-label">範囲</label>
      <input type="number" class="form-control" name="light_radius" value="{{if isNotNil .Member.LightSource}}{{.Member.LightSource.Radius}}{{else}}4{{end}}">
    </div>
    <div class="col-md-2 d-flex align-items-end">
      <div class="form-check">
        <input class="form-check-input" type="checkbox" name="light_enabled" id="light-en-{{.Index}}" {{if isNotNil .Member.LightSource}}{{if .Member.LightSource.Enabled}}checked{{end}}{{end}}>
        <label class="form-check-label" for="light-en-{{.Index}}">有効</label>
      </div>
    </div>
    <div class="col-md-2">
      <label class="form-label">R</label>
      <input type="number" class="form-control" name="light_r" min="0" max="255" value="{{if isNotNil .Member.LightSource}}{{.Member.LightSource.Color.R}}{{else}}255{{end}}">
    </div>
    <div class="col-md-2">
      <label class="form-label">G</label>
      <input type="number" class="form-control" name="light_g" min="0" max="255" value="{{if isNotNil .Member.LightSource}}{{.Member.LightSource.Color.G}}{{else}}255{{end}}">
    </div>
    <div class="col-md-2">
      <label class="form-label">B</label>
      <input type="number" class="form-control" name="light_b" min="0" max="255" value="{{if isNotNil .Member.LightSource}}{{.Member.LightSource.Color.B}}{{else}}220{{end}}">
    </div>
    <div class="col-md-2">
      <label class="form-label">A</label>
      <input type="number" class="form-control" name="light_a" min="0" max="255" value="{{if isNotNil .Member.LightSource}}{{.Member.LightSource.Color.A}}{{else}}255{{end}}">
    </div>
  </div>

  <h6 class="mt-4 mb-2">
    <input class="form-check-input me-1" type="checkbox" name="has_dialog" id="dialog-{{.Index}}" {{if isNotNil .Member.Dialog}}checked{{end}}
           onchange="document.getElementById('dialog-fields-{{.Index}}').classList.toggle('d-none', !this.checked)">
    <label class="form-check-label" for="dialog-{{.Index}}">会話</label>
  </h6>
  <div id="dialog-fields-{{.Index}}" class="row g-3 mb-3 {{if not (isNotNil .Member.Dialog)}}d-none{{end}}">
    <div class="col-md-6">
      <label class="form-label">メッセージキー</label>
      <input type="text" class="form-control" name="dialog_message_key" value="{{if isNotNil .Member.Dialog}}{{.Member.Dialog.MessageKey}}{{end}}">
    </div>
  </div>

  <button type="submit" class="btn btn-success mt-3">保存</button>
</form>
{{end}}

{{define "recipes"}}
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Ruins Editor - Recipes</title>
  {{template "common-head"}}
  <style>
    .sidebar { width: 280px; min-width: 280px; overflow-y: auto; }
    .recipe-entry { cursor: pointer; padding: 4px 8px; border-bottom: 1px solid #dee2e6; display: flex; align-items: center; gap: 6px; font-size: 13px; }
    .recipe-entry:hover { background: rgba(0,0,0,0.04); }
    .recipe-entry.active { background: rgba(13,110,253,0.25); border-left: 3px solid #0d6efd; }
    .main-content { flex: 1; overflow-y: auto; padding: 24px; }
    .content-area { height: calc(100vh - 40px); }
  </style>
</head>
<body style="overflow:hidden;">
  {{template "header" .}}
  <div class="d-flex content-area">
    <div class="sidebar border-end p-0 d-flex flex-column">
      <div class="p-2 border-bottom">
        <form hx-post="/recipes/new" hx-target="#recipe-edit-panel" hx-swap="innerHTML" class="d-flex gap-1">
          <input type="text" class="form-control form-control-sm" name="name" required placeholder="新規レシピ">
          <button type="submit" class="btn btn-primary btn-sm">追加</button>
        </form>
      </div>
      <div id="recipe-count" class="p-1 border-bottom text-secondary" style="font-size:12px;">
        {{len .Items}} recipes
      </div>
      <div id="recipe-list" style="overflow-y:auto;flex:1;">
        {{range .Items}}
        {{template "recipe-entry" .}}
        {{end}}
      </div>
    </div>
    <div class="main-content" id="recipe-edit-panel">
      {{if .Edit}}
      {{template "recipe-edit" .Edit}}
      {{else}}
      <div class="text-secondary mt-5 text-center">レシピを選択してください</div>
      {{end}}
    </div>
  </div>
  {{template "tom-select-init"}}
</body>
</html>
{{end}}

{{define "recipe-entry"}}
<div class="recipe-entry{{if .Active}} active{{end}}" id="rentry-{{.Index}}"
     hx-get="/recipes/{{.Index}}/edit" hx-target="#recipe-edit-panel" hx-swap="innerHTML"
     onclick="document.querySelectorAll('.recipe-entry').forEach(e=>e.classList.remove('active'));this.classList.add('active');">
  <span class="text-truncate flex-grow-1">{{.Recipe.Name}}</span>
  <span class="badge text-bg-secondary">{{len .Recipe.Inputs}}素材</span>
</div>
{{end}}

{{define "recipe-list-oob"}}
<div id="recipe-list" hx-swap-oob="innerHTML:#recipe-list">
{{range .Items}}
{{template "recipe-entry" .}}
{{end}}
</div>
{{end}}

{{define "recipe-count-oob"}}
<div id="recipe-count" hx-swap-oob="innerHTML:#recipe-count">
  {{len .Items}} recipes
</div>
{{end}}

{{define "recipe-edit"}}
<script>
var recipeInputIdx = {{len .Recipe.Inputs}};
function addRecipeInput() {
  var container = document.getElementById('recipe-inputs');
  var div = document.createElement('div');
  div.className = 'row g-2 mb-2 recipe-input-row';
  div.innerHTML = '<div class="col-md-6"><select class="form-select" name="input_name_' + recipeInputIdx + '"><option value="">-- 選択 --</option>' +
    document.getElementById('item-options-template').innerHTML +
    '</select></div><div class="col-md-3"><input type="number" class="form-control" name="input_amount_' + recipeInputIdx + '" value="1" min="1"></div>' +
    '<div class="col-md-3"><button type="button" class="btn btn-outline-danger btn-sm" onclick="this.closest(\'.recipe-input-row\').remove()">削除</button></div>';
  container.appendChild(div);
  initTomSelect(div);
  recipeInputIdx++;
}
</script>
<template id="item-options-template">
  {{range .ItemOptions}}<option value="{{.Name}}">{{.Label}}</option>{{end}}
</template>

<form hx-post="/recipes/{{.Index}}" hx-target="#recipe-edit-panel" hx-swap="innerHTML">
  <div class="d-flex align-items-center gap-3 mb-3">
    <h5 class="mb-0 me-auto">{{.Recipe.Name}}</h5>
    <button class="btn btn-outline-danger btn-sm" type="button" hx-delete="/recipes/{{.Index}}" hx-target="#recipe-edit-panel" hx-swap="innerHTML" hx-confirm="削除しますか?">削除</button>
  </div>

  <div class="row g-3 mb-3">
    <div class="col-md-4">
      <label class="form-label">成果物</label>
      <select class="form-select" name="name">
        <option value="">-- 選択 --</option>
        {{range .ItemOptions}}
        <option value="{{.Name}}" {{selected $.Recipe.Name .Name}}>{{.Label}}</option>
        {{end}}
      </select>
    </div>
  </div>

  <h6 class="mb-2">素材</h6>
  <div id="recipe-inputs">
    {{range $i, $input := .Recipe.Inputs}}
    <div class="row g-2 mb-2 recipe-input-row">
      <div class="col-md-6">
        <select class="form-select" name="input_name_{{$i}}">
          <option value="">-- 選択 --</option>
          {{range $.ItemOptions}}
          <option value="{{.Name}}" {{selected $input.Name .Name}}>{{.Label}}</option>
          {{end}}
        </select>
      </div>
      <div class="col-md-3">
        <input type="number" class="form-control" name="input_amount_{{$i}}" value="{{$input.Amount}}" min="1">
      </div>
      <div class="col-md-3">
        <button type="button" class="btn btn-outline-danger btn-sm" onclick="this.closest('.recipe-input-row').remove()">削除</button>
      </div>
    </div>
    {{end}}
  </div>
  <button type="button" class="btn btn-outline-secondary btn-sm mb-3" onclick="addRecipeInput()">+ 素材を追加</button>

  <div>
    <button type="submit" class="btn btn-success">保存</button>
  </div>
</form>
{{end}}

{{define "cutter"}}
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Ruins Editor - Sprite Cutter</title>
  {{template "common-head"}}
  <style>
    .cell-grid {
      display: inline-grid;
      gap: 1px;
      background: #333;
      border: 1px solid #555;
    }
    .cell {
      position: relative;
      overflow: hidden;
      image-rendering: pixelated;
    }
    .cell img {
      position: absolute;
      image-rendering: pixelated;
    }
    .cell-overlay {
      position: absolute;
      bottom: 0;
      left: 0;
      right: 0;
      background: rgba(0,0,0,0.7);
      opacity: 0;
      transition: opacity 0.15s;
    }
    .cell:hover .cell-overlay { opacity: 1; }
    .cell.named { outline: 2px solid #0d6efd; outline-offset: -2px; }
    .cell-name-input {
      width: 100%;
      background: transparent;
      border: none;
      color: #fff;
      font-size: 12px;
      padding: 4px 5px;
      outline: none;
    }
  </style>
</head>
<body style="overflow:hidden;">
  {{template "header" .}}
  <div style="height:calc(100vh - 40px);overflow-y:auto;padding:24px;">
  <h1 class="mb-3">Sprite Cutter</h1>
  <p class="text-secondary">256x256のスプライトシートPNGをアップロードし、32x32の個別スプライトに切り出して名前をつけて保存します。</p>

  <form hx-post="/cutter/upload" hx-target="body" hx-swap="innerHTML" hx-encoding="multipart/form-data" class="mb-4">
    <div class="row g-2 align-items-end">
      <div class="col-auto">
        <label class="form-label">スプライトシートPNG</label>
        <input type="file" class="form-control" name="sheet" accept="image/png" required>
      </div>
      <div class="col-auto">
        <button type="submit" class="btn btn-primary">アップロード</button>
      </div>
    </div>
  </form>

  {{if .Uploaded}}
  <div id="save-result"></div>
  <form hx-post="/cutter/save" hx-target="#save-result" hx-swap="innerHTML" class="mb-4">
    <div class="d-flex align-items-center gap-3 mb-3">
      <h5 class="mb-0">{{.Cols}}x{{.Rows}} = {{len .Cells}} セル ({{.CellSize}}x{{.CellSize}}px)</h5>
      <button type="submit" class="btn btn-success">選択したスプライトを保存</button>
      <button type="button" class="btn btn-outline-secondary btn-sm" onclick="autoName()">連番で命名</button>
    </div>
    <div class="cell-grid" style="grid-template-columns: repeat({{.Cols}}, {{.CellSize | mul 2}}px);">
      {{range .Cells}}
      <div class="cell" id="cell-{{.Index}}" style="width:{{$.CellSize | mul 2}}px;height:{{$.CellSize | mul 2}}px;background:#222;">
        <img src="/cutter/preview" style="width:{{$.Cols | mul $.CellSize | mul 2}}px;height:{{$.Rows | mul $.CellSize | mul 2}}px;left:-{{.Col | mul $.CellSize | mul 2}}px;top:-{{.Row | mul $.CellSize | mul 2}}px;">
        <div class="cell-overlay">
          <input type="text" class="cell-name-input" name="name_{{.Index}}" placeholder="名前" oninput="toggleNamed(this)">
        </div>
      </div>
      {{end}}
    </div>
  </form>
  <script>
  function toggleNamed(input) {
    var cell = input.closest('.cell');
    if (input.value.trim()) {
      cell.classList.add('named');
    } else {
      cell.classList.remove('named');
    }
  }
  function autoName() {
    document.querySelectorAll('.cell-name-input').forEach(function(input, i) {
      if (!input.value) {
        input.value = 'sprite_' + i;
        toggleNamed(input);
      }
    });
  }
  </script>
  {{end}}
  </div>
</body>
</html>
{{end}}

{{define "sprite-picker-js"}}
<script>
function pickSprite(el) {
  document.querySelectorAll('.sprite-option').forEach(function(e) { e.classList.remove('border-primary', 'bg-primary', 'bg-opacity-25'); });
  el.classList.add('border-primary', 'bg-primary', 'bg-opacity-25');
  var key = el.getAttribute('data-key');
  document.getElementById('sprite-key-input').value = key;
  document.getElementById('sprite-key-display').value = key;
  var preview = document.getElementById('sprite-preview');
  if (preview) { preview.style.cssText = el.querySelector('span').style.cssText; }
  document.getElementById('sprite-picker-panel').classList.add('d-none');
}
function openSpritePicker() {
  var sheet = document.querySelector('[name="sprite_sheet_name"]').value;
  if (!sheet) return;
  var panel = document.getElementById('sprite-picker-panel');
  var grid = document.getElementById('sprite-key-grid');
  if (grid.children.length === 0 || grid.getAttribute('data-sheet') !== sheet) {
    grid.setAttribute('data-sheet', sheet);
    htmx.ajax('GET', '/sprites/' + sheet + '/keys', {target:'#sprite-key-grid', swap:'innerHTML'}).then(function() {
      highlightCurrent();
    });
  }
  panel.classList.remove('d-none');
  var search = document.getElementById('sprite-search');
  search.value = '';
  search.focus();
  filterSprites('');
}
function highlightCurrent() {
  var current = document.getElementById('sprite-key-input').value;
  if (!current) return;
  document.querySelectorAll('.sprite-option').forEach(function(el) {
    if (el.getAttribute('data-key') === current) {
      el.classList.add('border-primary', 'bg-primary', 'bg-opacity-25');
    }
  });
}
function filterSprites(query) {
  var q = query.toLowerCase();
  document.querySelectorAll('.sprite-option').forEach(function(el) {
    var key = el.getAttribute('data-key').toLowerCase();
    el.style.display = key.indexOf(q) !== -1 ? '' : 'none';
  });
}
document.addEventListener('click', function(e) {
  var panel = document.getElementById('sprite-picker-panel');
  if (panel && !panel.contains(e.target) && e.target.id !== 'sprite-key-display') {
    panel.classList.add('d-none');
  }
});
</script>
{{end}}

{{define "item-edit"}}
{{$m := melee .Item}}
{{$f := fire .Item}}
{{$c := consumable .Item}}
{{$w := wearable .Item}}
{{template "sprite-picker-js"}}

<form hx-post="/items/{{.Index}}" hx-target="#edit-panel" hx-swap="innerHTML">
  <div class="d-flex align-items-center gap-3 mb-3">
    <span id="sprite-preview" style="{{spriteStyle .Item.SpriteSheetName .Item.SpriteKey 2}}"></span>
    <h5 class="mb-0 me-auto">{{.Item.Name}}</h5>
    <button class="btn btn-outline-danger btn-sm" type="button" hx-delete="/items/{{.Index}}" hx-target="#edit-panel" hx-swap="innerHTML" hx-confirm="削除しますか?">削除</button>
  </div>

  <div class="row g-3 mb-3">
    <div class="col-md-3">
      <label class="form-label">名前</label>
      <input type="text" class="form-control" name="name" value="{{.Item.Name}}" required>
    </div>
    <div class="col-md-5">
      <label class="form-label">説明</label>
      <input type="text" class="form-control" name="description" value="{{.Item.Description}}">
    </div>
    <div class="col-md-2">
      <label class="form-label">SpriteSheet</label>
      <select class="form-select" name="sprite_sheet_name"
              onchange="document.getElementById('sprite-key-grid').innerHTML='';document.getElementById('sprite-key-grid').removeAttribute('data-sheet');document.getElementById('sprite-key-input').value='';document.getElementById('sprite-key-display').value='';">
        <option value="">-- 選択 --</option>
        {{range .SheetNames}}
        <option value="{{.}}" {{selected $.Item.SpriteSheetName .}}>{{.}}</option>
        {{end}}
      </select>
    </div>
    <div class="col-md-2 position-relative">
      <label class="form-label">SpriteKey</label>
      <input type="hidden" name="sprite_key" id="sprite-key-input" value="{{.Item.SpriteKey}}">
      <input type="text" class="form-control" id="sprite-key-display" value="{{.Item.SpriteKey}}" readonly onclick="openSpritePicker()" style="cursor:pointer;" placeholder="クリックで選択">
      <div id="sprite-picker-panel" class="d-none position-absolute bg-body border rounded shadow p-2 mt-1" style="z-index:1050;width:400px;right:0;">
        <input type="text" class="form-control form-control-sm mb-2" id="sprite-search" placeholder="検索..." oninput="filterSprites(this.value)">
        <div id="sprite-key-grid" class="d-flex flex-wrap gap-1" style="max-height:200px;overflow-y:auto;"></div>
      </div>
    </div>
  </div>

  <div class="row g-3 mb-3">
    <div class="col-md-2">
      <label class="form-label">価値</label>
      <input type="number" class="form-control" name="value" value="{{.Item.Value}}" required>
    </div>
    <div class="col-md-2">
      <label class="form-label">重量</label>
      <input type="number" class="form-control" name="weight" step="0.01" value="{{if isNotNil .Item.Weight}}{{derefFloat .Item.Weight}}{{end}}">
    </div>
    <div class="col-md-2">
      <label class="form-label">攻撃力</label>
      <input type="number" class="form-control" name="inflicts_damage" value="{{if isNotNil .Item.InflictsDamage}}{{derefInt .Item.InflictsDamage}}{{end}}">
    </div>
    <div class="col-md-2">
      <label class="form-label">栄養</label>
      <input type="number" class="form-control" name="provides_nutrition" value="{{if isNotNil .Item.ProvidesNutrition}}{{derefInt .Item.ProvidesNutrition}}{{end}}">
    </div>
    <div class="col-md-2 d-flex align-items-end">
      <div class="form-check">
        <input type="checkbox" class="form-check-input" name="stackable" id="stackable-{{.Index}}" {{if derefBool .Item.Stackable}}checked{{end}}>
        <label class="form-check-label" for="stackable-{{.Index}}">スタック可能</label>
      </div>
    </div>
  </div>

  <div class="card mb-3">
    <div class="card-header">
      <div class="form-check">
        <input type="checkbox" class="form-check-input" name="has_consumable" id="cons-{{.Index}}" {{if isNotNil .Item.Consumable}}checked{{end}}>
        <label class="form-check-label" for="cons-{{.Index}}">消費アイテム</label>
      </div>
    </div>
    <div class="card-body">
      <div class="row g-3">
        <div class="col-md-4">
          <label class="form-label">使用場面</label>
          {{template "select-usable-scene" (selectData "consumable_usable_scene" $c.UsableScene)}}
        </div>
        <div class="col-md-4">
          <label class="form-label">対象グループ</label>
          {{template "select-target-group" (selectData "consumable_target_group" $c.TargetGroup)}}
        </div>
        <div class="col-md-4">
          <label class="form-label">対象数</label>
          {{template "select-target-num" (selectData "consumable_target_num" $c.TargetNum)}}
        </div>
      </div>
    </div>
  </div>

  <div class="card mb-3">
    <div class="card-header">
      <div class="form-check">
        <input type="checkbox" class="form-check-input" name="has_melee" id="melee-{{.Index}}" {{if isNotNil .Item.Melee}}checked{{end}}>
        <label class="form-check-label" for="melee-{{.Index}}">近接攻撃</label>
      </div>
    </div>
    <div class="card-body">
      <div class="row g-3 mb-2">
        <div class="col-md-2">
          <label class="form-label">命中</label>
          <input type="number" class="form-control" name="melee_accuracy" value="{{$m.Accuracy}}">
        </div>
        <div class="col-md-2">
          <label class="form-label">ダメージ</label>
          <input type="number" class="form-control" name="melee_damage" value="{{$m.Damage}}">
        </div>
        <div class="col-md-2">
          <label class="form-label">回数</label>
          <input type="number" class="form-control" name="melee_attack_count" value="{{$m.AttackCount}}">
        </div>
        <div class="col-md-2">
          <label class="form-label">コスト</label>
          <input type="number" class="form-control" name="melee_cost" value="{{$m.Cost}}">
        </div>
        <div class="col-md-2">
          <label class="form-label">属性</label>
          {{template "select-element" (selectData "melee_element" $m.Element)}}
        </div>
        <div class="col-md-2">
          <label class="form-label">種別</label>
          {{template "select-attack-category" (selectData "melee_attack_category" $m.AttackCategory)}}
        </div>
      </div>
      <div class="row g-3">
        <div class="col-md-2">
          <label class="form-label">対象グループ</label>
          {{template "select-target-group" (selectData "melee_target_group" $m.TargetGroup)}}
        </div>
        <div class="col-md-2">
          <label class="form-label">対象数</label>
          {{template "select-target-num" (selectData "melee_target_num" $m.TargetNum)}}
        </div>
      </div>
    </div>
  </div>

  <div class="card mb-3">
    <div class="card-header">
      <div class="form-check">
        <input type="checkbox" class="form-check-input" name="has_fire" id="fire-{{.Index}}" {{if isNotNil .Item.Fire}}checked{{end}}>
        <label class="form-check-label" for="fire-{{.Index}}">射撃</label>
      </div>
    </div>
    <div class="card-body">
      <div class="row g-3 mb-2">
        <div class="col-md-2">
          <label class="form-label">命中</label>
          <input type="number" class="form-control" name="fire_accuracy" value="{{$f.Accuracy}}">
        </div>
        <div class="col-md-2">
          <label class="form-label">ダメージ</label>
          <input type="number" class="form-control" name="fire_damage" value="{{$f.Damage}}">
        </div>
        <div class="col-md-2">
          <label class="form-label">回数</label>
          <input type="number" class="form-control" name="fire_attack_count" value="{{$f.AttackCount}}">
        </div>
        <div class="col-md-2">
          <label class="form-label">コスト</label>
          <input type="number" class="form-control" name="fire_cost" value="{{$f.Cost}}">
        </div>
        <div class="col-md-2">
          <label class="form-label">属性</label>
          {{template "select-element" (selectData "fire_element" $f.Element)}}
        </div>
        <div class="col-md-2">
          <label class="form-label">種別</label>
          {{template "select-attack-category" (selectData "fire_attack_category" $f.AttackCategory)}}
        </div>
      </div>
      <div class="row g-3">
        <div class="col-md-2">
          <label class="form-label">弾倉</label>
          <input type="number" class="form-control" name="fire_magazine_size" value="{{$f.MagazineSize}}">
        </div>
        <div class="col-md-2">
          <label class="form-label">装填工数</label>
          <input type="number" class="form-control" name="fire_reload_effort" value="{{$f.ReloadEffort}}">
        </div>
        <div class="col-md-2">
          <label class="form-label">弾薬タグ</label>
          <input type="text" class="form-control" name="fire_ammo_tag" value="{{$f.AmmoTag}}">
        </div>
        <div class="col-md-2">
          <label class="form-label">対象グループ</label>
          {{template "select-target-group" (selectData "fire_target_group" $f.TargetGroup)}}
        </div>
        <div class="col-md-2">
          <label class="form-label">対象数</label>
          {{template "select-target-num" (selectData "fire_target_num" $f.TargetNum)}}
        </div>
      </div>
    </div>
  </div>

  <div class="card mb-3">
    <div class="card-header">
      <div class="form-check">
        <input type="checkbox" class="form-check-input" name="has_wearable" id="wear-{{.Index}}" {{if isNotNil .Item.Wearable}}checked{{end}}>
        <label class="form-check-label" for="wear-{{.Index}}">防具</label>
      </div>
    </div>
    <div class="card-body">
      <div class="row g-3">
        <div class="col-md-3">
          <label class="form-label">防御力</label>
          <input type="number" class="form-control" name="wearable_defense" value="{{$w.Defense}}">
        </div>
        <div class="col-md-3">
          <label class="form-label">装備種別</label>
          {{template "select-equipment-category" (selectData "wearable_equipment_category" $w.EquipmentCategory)}}
        </div>
        <div class="col-md-3">
          <label class="form-label">耐寒</label>
          <input type="number" class="form-control" name="wearable_insulation_cold" value="{{$w.InsulationCold}}">
        </div>
        <div class="col-md-3">
          <label class="form-label">耐暑</label>
          <input type="number" class="form-control" name="wearable_insulation_heat" value="{{$w.InsulationHeat}}">
        </div>
      </div>
    </div>
  </div>

  <button type="submit" class="btn btn-primary">保存</button>
</form>
{{end}}
`
