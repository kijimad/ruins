package editor

var templateText = `
{{define "index"}}
<!DOCTYPE html>
<html lang="ja" data-bs-theme="dark">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Ruins Editor</title>
  <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet">
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
</head>
<body class="p-4">
  <nav class="mb-3">
    <a href="/" class="btn btn-outline-light btn-sm me-2">Items</a>
    <a href="/cutter" class="btn btn-outline-light btn-sm">Sprite Cutter</a>
  </nav>
  <h1 class="mb-3">Ruins Editor - Items</h1>
  <p class="text-secondary">{{len .Items}} items</p>

  <form class="row g-2 align-items-end mb-4" hx-post="/items/new" hx-target="body" hx-swap="innerHTML">
    <div class="col-auto">
      <label class="form-label">名前</label>
      <input type="text" class="form-control" name="name" required placeholder="新しいアイテム名">
    </div>
    <div class="col-auto">
      <label class="form-label">説明</label>
      <input type="text" class="form-control" name="description" placeholder="説明">
    </div>
    <div class="col-auto">
      <button type="submit" class="btn btn-primary">追加</button>
    </div>
  </form>

  <script>
  function closeCurrentEdit() {
    var editingTr = document.querySelector('tr:has(form)');
    if (editingTr) {
      var idx = editingTr.id.replace('item-', '');
      htmx.ajax('GET', '/items/' + idx, {target: editingTr, swap: 'outerHTML'});
    }
  }
  </script>
  <table class="table table-hover">
    <thead>
      <tr>
        <th>#</th>
        <th>名前</th>
        <th>説明</th>
        <th>種別</th>
        <th>価値</th>
        <th>重量</th>
        <th></th>
      </tr>
    </thead>
    <tbody id="item-list">
      {{range .Items}}
      {{template "item-row" .}}
      {{end}}
    </tbody>
  </table>
</body>
</html>
{{end}}

{{define "item-row"}}
<tr id="item-{{.Index}}" style="cursor:pointer;" hx-get="/items/{{.Index}}/edit" hx-target="#item-{{.Index}}" hx-swap="outerHTML" onclick="closeCurrentEdit()">
  <td>{{.Index}}</td>
  <td><span style="{{spriteStyle .Item.SpriteSheetName .Item.SpriteKey 1}}" class="me-1 align-middle"></span>{{.Item.Name}}</td>
  <td class="text-truncate" style="max-width:300px;">{{.Item.Description}}</td>
  <td>
    {{if isNotNil .Item.Weapon}}<span class="badge text-bg-primary">武器</span>{{end}}
    {{if isNotNil .Item.Wearable}}<span class="badge text-bg-info">防具</span>{{end}}
    {{if isNotNil .Item.Consumable}}<span class="badge text-bg-success">消費</span>{{end}}
    {{if isNotNil .Item.Ammo}}<span class="badge text-bg-warning">弾薬</span>{{end}}
    {{if isNotNil .Item.Book}}<span class="badge text-bg-secondary">本</span>{{end}}
    {{if isNotNil .Item.Melee}}<span class="badge text-bg-danger">近接</span>{{end}}
    {{if isNotNil .Item.Fire}}<span class="badge text-bg-danger">射撃</span>{{end}}
  </td>
  <td>{{.Item.Value}}</td>
  <td>{{if isNotNil .Item.Weight}}{{derefFloat .Item.Weight}}{{end}}</td>
  <td>
    <button class="btn btn-outline-danger btn-sm" hx-delete="/items/{{.Index}}" hx-target="#item-{{.Index}}" hx-swap="outerHTML" hx-confirm="削除しますか?" onclick="event.stopPropagation();">削除</button>
  </td>
</tr>
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

{{define "cutter"}}
<!DOCTYPE html>
<html lang="ja" data-bs-theme="dark">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Ruins Editor - Sprite Cutter</title>
  <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet">
  <script src="https://unpkg.com/htmx.org@2.0.4"></script>
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
<body class="p-4">
  <nav class="mb-3">
    <a href="/" class="btn btn-outline-light btn-sm me-2">Items</a>
    <a href="/cutter" class="btn btn-outline-light btn-sm">Sprite Cutter</a>
  </nav>
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
</body>
</html>
{{end}}

{{define "item-edit"}}
{{$m := melee .Item}}
{{$f := fire .Item}}
{{$c := consumable .Item}}
{{$w := wearable .Item}}
<tr id="item-{{.Index}}">
  <td colspan="7">
    <form hx-post="/items/{{.Index}}" hx-target="#item-{{.Index}}" hx-swap="outerHTML">
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

      <div class="d-flex align-items-center gap-3 mb-3">
        <span id="sprite-preview" style="{{spriteStyle .Item.SpriteSheetName .Item.SpriteKey 2}}"></span>
        <h5 class="mb-0 me-auto">{{.Item.Name}} を編集</h5>
        <button type="button" class="btn btn-outline-secondary btn-sm" hx-get="/items/{{.Index}}" hx-target="#item-{{.Index}}" hx-swap="outerHTML">閉じる</button>
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

      <div class="d-flex gap-2">
        <button type="submit" class="btn btn-primary">保存</button>
        <button type="button" class="btn btn-secondary" hx-get="/items/{{.Index}}" hx-target="#item-{{.Index}}" hx-swap="outerHTML">キャンセル</button>
      </div>
    </form>
  </td>
</tr>
{{end}}
`
