package editor

var templateTextPalette = `
{{define "palettes"}}
<!DOCTYPE html>
<html lang="ja">
<head>
  <meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Ruins Editor - Palettes</title>
  {{template "common-head"}}{{template "sidebar-style"}}
</head>
<body style="overflow:hidden;">
  {{template "header" .}}
  <div class="d-flex content-area">
    <div class="sidebar border-end p-0 d-flex flex-column">
      <div class="p-2 border-bottom">
        <form hx-post="/palettes/new" hx-target="#pal-edit-panel" hx-swap="innerHTML" class="d-flex gap-1">
          <input type="text" class="form-control form-control-sm" name="id" required placeholder="新規パレットID">
          <button type="submit" class="btn btn-primary btn-sm">追加</button>
        </form>
      </div>
      <div id="pal-count" class="p-1 border-bottom text-secondary" style="font-size:12px;">{{len .Items}} palettes</div>
      <div id="pal-list" style="overflow-y:auto;flex:1;">
        {{range .Items}}{{template "pal-entry" .}}{{end}}
      </div>
    </div>
    <div class="main-content" id="pal-edit-panel">
      {{if .Edit}}{{template "palette-edit" .Edit}}{{else}}<div class="text-secondary mt-5 text-center">パレットを選択してください</div>{{end}}
    </div>
  </div>
  {{template "tom-select-init"}}
</body>
</html>
{{end}}

{{define "pal-entry"}}
<div class="sidebar-entry{{if .Active}} active{{end}}" hx-get="/palettes/{{.Palette.ID}}/edit" hx-target="#pal-edit-panel" hx-swap="innerHTML"
     onclick="document.querySelectorAll('.sidebar-entry').forEach(e=>e.classList.remove('active'));this.classList.add('active');">
  <span class="text-truncate flex-grow-1">{{.Palette.ID}}</span>
  <span class="badge text-bg-secondary">{{len .Palette.Terrain}}/{{len .Palette.Props}}/{{len .Palette.NPCs}}</span>
</div>
{{end}}

{{define "pal-list-oob"}}
<div id="pal-list" hx-swap-oob="innerHTML:#pal-list">{{range .Items}}{{template "pal-entry" .}}{{end}}</div>
{{end}}

{{define "pal-count-oob"}}
<div id="pal-count" hx-swap-oob="innerHTML:#pal-count">{{len .Items}} palettes</div>
{{end}}

{{define "palette-edit"}}
<script>
function addTerrainEntry() {
  var c = document.getElementById('terrain-entries');
  var div = document.createElement('div');
  div.className = 'row g-2 mb-2 mapping-row';
  div.innerHTML = '<div class="col-2"><input type="text" class="form-control form-control-sm font-monospace text-center" name="terrain_char[]" maxlength="1" placeholder="文字"></div><div class="col-8"><select class="form-select form-select-sm" name="terrain_value[]"><option value="">-- 選択 --</option>'+document.getElementById('tile-options-tpl').innerHTML+'</select></div><div class="col-2"><button type="button" class="btn btn-outline-danger btn-sm" onclick="this.closest(\'.mapping-row\').remove()">×</button></div>';
  c.appendChild(div);
}

function addPropEntry() {
  var c = document.getElementById('prop-entries');
  var div = document.createElement('div');
  div.className = 'row g-2 mb-2 mapping-row';
  div.innerHTML = '<div class="col-2"><input type="text" class="form-control form-control-sm font-monospace text-center" name="prop_char[]" maxlength="1" placeholder="文字"></div><div class="col-4"><select class="form-select form-select-sm" name="prop_value[]"><option value="">-- Prop --</option>'+document.getElementById('prop-options-tpl').innerHTML+'</select></div><div class="col-4"><select class="form-select form-select-sm" name="prop_tile[]"><option value="">-- タイル --</option>'+document.getElementById('tile-options-tpl').innerHTML+'</select></div><div class="col-2"><button type="button" class="btn btn-outline-danger btn-sm" onclick="this.closest(\'.mapping-row\').remove()">×</button></div>';
  c.appendChild(div);
}

function addNPCEntry() {
  var c = document.getElementById('npc-entries');
  var div = document.createElement('div');
  div.className = 'row g-2 mb-2 mapping-row';
  div.innerHTML = '<div class="col-2"><input type="text" class="form-control form-control-sm font-monospace text-center" name="npc_char[]" maxlength="1" placeholder="文字"></div><div class="col-4"><select class="form-select form-select-sm" name="npc_value[]"><option value="">-- NPC --</option>'+document.getElementById('npc-options-tpl').innerHTML+'</select></div><div class="col-4"><select class="form-select form-select-sm" name="npc_tile[]"><option value="">-- タイル --</option>'+document.getElementById('tile-options-tpl').innerHTML+'</select></div><div class="col-2"><button type="button" class="btn btn-outline-danger btn-sm" onclick="this.closest(\'.mapping-row\').remove()">×</button></div>';
  c.appendChild(div);
}
</script>

<template id="tile-options-tpl">{{range .TileNames}}<option value="{{.}}">{{.}}</option>{{end}}</template>
<template id="prop-options-tpl">{{range .PropNames}}<option value="{{.}}">{{.}}</option>{{end}}</template>
<template id="npc-options-tpl">{{range .NPCNames}}<option value="{{.}}">{{.}}</option>{{end}}</template>

<form hx-post="/palettes/{{.Palette.ID}}" hx-target="#pal-edit-panel" hx-swap="innerHTML">
  <div class="d-flex align-items-center gap-3 mb-3">
    <h5 class="mb-0 me-auto">{{.Palette.ID}}</h5>
    <button class="btn btn-outline-danger btn-sm" type="button" hx-delete="/palettes/{{.Palette.ID}}" hx-target="#pal-edit-panel" hx-swap="innerHTML" hx-confirm="削除しますか?">削除</button>
  </div>
  <div class="row g-3 mb-3">
    <div class="col-md-6">
      <label class="form-label">説明</label>
      <input type="text" class="form-control form-control-sm" name="description" value="{{.Palette.Description}}">
    </div>
  </div>

  <!-- 地形マッピング -->
  <h6 class="mb-2">地形 (文字 → タイル)</h6>
  <div id="terrain-entries">
    {{range $e := .TerrainEntries}}
    <div class="row g-2 mb-2 mapping-row">
      <div class="col-2">
        <input type="text" class="form-control form-control-sm font-monospace text-center" name="terrain_char[]" value="{{$e.Char}}" maxlength="1">
      </div>
      <div class="col-8">
        <select class="form-select form-select-sm" name="terrain_value[]">
          <option value="">-- 選択 --</option>
          {{range $.TileNames}}<option value="{{.}}" {{if eq . $e.Value}}selected{{end}}>{{.}}</option>{{end}}
        </select>
      </div>
      <div class="col-2">
        <button type="button" class="btn btn-outline-danger btn-sm" onclick="this.closest('.mapping-row').remove()">×</button>
      </div>
    </div>
    {{end}}
  </div>
  <button type="button" class="btn btn-outline-secondary btn-sm mb-3" onclick="addTerrainEntry()">+ 地形を追加</button>

  <!-- Propマッピング -->
  <h6 class="mb-2">Props (文字 → Prop / タイル)</h6>
  <div id="prop-entries">
    {{range $e := .PropEntries}}
    <div class="row g-2 mb-2 mapping-row">
      <div class="col-2">
        <input type="text" class="form-control form-control-sm font-monospace text-center" name="prop_char[]" value="{{$e.Char}}" maxlength="1">
      </div>
      <div class="col-4">
        <select class="form-select form-select-sm" name="prop_value[]">
          <option value="">-- Prop --</option>
          {{range $.PropNames}}<option value="{{.}}" {{if eq . $e.Value}}selected{{end}}>{{.}}</option>{{end}}
        </select>
      </div>
      <div class="col-4">
        <select class="form-select form-select-sm" name="prop_tile[]">
          <option value="">-- タイル --</option>
          {{range $.TileNames}}<option value="{{.}}" {{if eq . $e.Tile}}selected{{end}}>{{.}}</option>{{end}}
        </select>
      </div>
      <div class="col-2">
        <button type="button" class="btn btn-outline-danger btn-sm" onclick="this.closest('.mapping-row').remove()">×</button>
      </div>
    </div>
    {{end}}
  </div>
  <button type="button" class="btn btn-outline-secondary btn-sm mb-3" onclick="addPropEntry()">+ Propを追加</button>

  <!-- NPCマッピング -->
  <h6 class="mb-2">NPCs (文字 → NPC / タイル)</h6>
  <div id="npc-entries">
    {{range $e := .NPCEntries}}
    <div class="row g-2 mb-2 mapping-row">
      <div class="col-2">
        <input type="text" class="form-control form-control-sm font-monospace text-center" name="npc_char[]" value="{{$e.Char}}" maxlength="1">
      </div>
      <div class="col-4">
        <select class="form-select form-select-sm" name="npc_value[]">
          <option value="">-- NPC --</option>
          {{range $.NPCNames}}<option value="{{.}}" {{if eq . $e.Value}}selected{{end}}>{{.}}</option>{{end}}
        </select>
      </div>
      <div class="col-4">
        <select class="form-select form-select-sm" name="npc_tile[]">
          <option value="">-- タイル --</option>
          {{range $.TileNames}}<option value="{{.}}" {{if eq . $e.Tile}}selected{{end}}>{{.}}</option>{{end}}
        </select>
      </div>
      <div class="col-2">
        <button type="button" class="btn btn-outline-danger btn-sm" onclick="this.closest('.mapping-row').remove()">×</button>
      </div>
    </div>
    {{end}}
  </div>
  <button type="button" class="btn btn-outline-secondary btn-sm mb-3" onclick="addNPCEntry()">+ NPCを追加</button>

  <div><button type="submit" class="btn btn-success">保存</button></div>
</form>
{{end}}
`
