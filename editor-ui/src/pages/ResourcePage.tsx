import { useEffect, useMemo, useRef, useState } from "react";
import {
  Box,
  Flex,
  Heading,
  Text,
  Button,
  Input,
  Stack,
  Badge,
  Fieldset,
  NativeSelectRoot,
  NativeSelectField,
} from "@chakra-ui/react";
import { Switch } from "../components/Switch";
import { SpriteSelect } from "../components/SpriteSelect";
import { SearchableSelect } from "../components/SearchableSelect";
import {
  AttackCategory, Element, TargetGroup, TargetNum, UsableScene,
  EquipmentCategory, EquipSlot, AmmoTag, FactionMemberType,
  HealingValueType, SpriteDepth, ShelterType, WaterType, FoliageType,
} from "../oapi";
import {
  useResourceList,
  useResourceUpdate,
  useResourceCreate,
  useResourceDelete,
} from "../hooks/useResource";

type JsonValue = string | number | boolean | null | JsonValue[] | { [key: string]: JsonValue };

// リソースごとのオプショナルセクションとデフォルト値の定義
const optionalSections: Record<string, Record<string, JsonValue>> = {
  items: {
    weapon: {},
    melee: {
      accuracy: 100, attackCategory: "FIST", attackCount: 1, cost: 2,
      damage: 5, element: "NONE", targetGroup: "ENEMY", targetNum: "SINGLE",
    },
    fire: {
      accuracy: 90, ammoTag: "9mm", attackCategory: "CANON", attackCount: 1,
      cost: 4, damage: 10, element: "NONE", magazineSize: 1, reloadEffort: 10,
      targetGroup: "ENEMY", targetNum: "SINGLE",
    },
    wearable: {
      defense: 0, equipmentCategory: "TORSO", insulationCold: 0, insulationHeat: 0,
    },
    equipBonus: {
      agility: 0, dexterity: 0, sensation: 0, strength: 0, vitality: 0,
    },
    consumable: {
      targetGroup: "ALLY", targetNum: "SINGLE", usableScene: "ANY",
    },
    providesHealing: {
      amount: 0, ratio: 0.5, valueType: "PERCENTAGE",
    },
    ammo: {
      accuracyBonus: 0, ammoTag: "9mm", damageBonus: 0,
    },
    book: {
      totalEffort: 200,
      skill: { maxLevel: 2, requiredLevel: 0, targetSkill: "rifle" },
    },
  },
  members: {
    abilities: {
      agility: 5, defense: 0, dexterity: 5, sensation: 5, strength: 5, vitality: 5,
    },
    lightSource: {
      enabled: true, radius: 4,
      color: { r: 255, g: 255, b: 220, a: 255 },
    },
    dialog: {
      messageKey: "",
    },
  },
  props: {
    door: {},
    lightSource: {
      enabled: true, radius: 4,
      color: { r: 255, g: 255, b: 220, a: 255 },
    },
    doorLockTrigger: {},
    dungeonGateTrigger: {},
    warpNextTrigger: {},
    warpEscapeTrigger: {},
  },
  professions: {
    skills: [],
  },
};

// リソースごとの新規作成テンプレート。全フィールドを含めないと入力欄が出ない
const createTemplates: Record<string, Record<string, JsonValue>> = {
  items: {
    name: "新規", description: "", spriteKey: "field_item",
    spriteSheetName: "field", value: 0, weight: 0,
  },
  members: {
    name: "新規", spriteKey: "slime_0", spriteSheetName: "field",
    animKeys: ["slime_0", "slime_1"], commandTableName: "", dropTableName: "",
    factionType: "", isBoss: false, player: false,
  },
  recipes: {
    name: "新規", inputs: [],
  },
  "command-tables": {
    name: "新規", entries: [],
  },
  "drop-tables": {
    name: "新規", entries: [],
  },
  "item-tables": {
    name: "新規", entries: [],
  },
  "enemy-tables": {
    name: "新規", entries: [],
  },
  tiles: {
    name: "新規", description: "", blockPass: false, blockView: false,
    foliage: 0, shelter: 0, water: 0,
    spriteRender: { depth: 0, spriteKey: "dirt", spriteSheetName: "tile" },
  },
  props: {
    name: "新規", description: "", blockPass: false, blockView: false,
    spriteRender: { depth: 0, spriteKey: "field_item", spriteSheetName: "field" },
  },
  professions: {
    id: "new", name: "新規", description: "",
    abilities: { agility: 5, defense: 5, dexterity: 5, sensation: 5, strength: 5, vitality: 5 },
    items: [], equips: [],
  },
};

// 配列要素の新規追加テンプレート。フィールド名からデフォルト値を決定する
const arrayElementTemplates: Record<string, JsonValue> = {
  inputs: { name: "", amount: 1 },
  animKeys: "",
  entries: {}, // entriesはリソースごとに異なるため既存要素から推論する
  items: { name: "", count: 1 },
  equips: { name: "", slot: "" },
  skills: { id: "", value: 1 },
};

// entriesの各リソース用テンプレート
const entriesTemplates: Record<string, JsonValue> = {
  "command-tables": { weapon: "", weight: 1 },
  "drop-tables": { material: "", weight: 1 },
  "item-tables": { itemName: "", minDepth: 1, maxDepth: 10, weight: 1 },
  "enemy-tables": { enemyName: "", minDepth: 1, maxDepth: 10, weight: 1 },
};

// OAPIから生成されたenumの値を文字列配列として取得するヘルパー
function enumValues<T extends Record<string, string | number>>(obj: T): string[] {
  return Object.values(obj).map(String);
}

// 選択式フィールドの定義。OAPIから生成されたenum値を使用する
// allowEmpty が true の場合、空文字列の選択肢を先頭に追加する
const selectFieldOptions: Record<string, { options: string[]; allowEmpty?: boolean }> = {
  spriteSheetName: { options: ["field", "tile", "bg"] },
  factionType: { options: enumValues(FactionMemberType), allowEmpty: true },
  attackCategory: { options: enumValues(AttackCategory) },
  element: { options: enumValues(Element) },
  targetGroup: { options: enumValues(TargetGroup) },
  targetNum: { options: enumValues(TargetNum) },
  usableScene: { options: enumValues(UsableScene) },
  equipmentCategory: { options: enumValues(EquipmentCategory) },
  valueType: { options: enumValues(HealingValueType) },
  slot: { options: enumValues(EquipSlot) },
  ammoTag: { options: enumValues(AmmoTag), allowEmpty: true },
};

// 数値enum型のフィールド。OAPIから生成されたenum値を使用する
const numericSelectLabels: Record<string, Record<number, string>> = {
  foliage: { [FoliageType.NUMBER_0]: "なし", [FoliageType.NUMBER_MINUS_1]: "草原", [FoliageType.NUMBER_MINUS_3]: "森" },
  shelter: { [ShelterType.NUMBER_0]: "屋外", [ShelterType.NUMBER_5]: "半屋外", [ShelterType.NUMBER_10]: "屋内" },
  water: { [WaterType.NUMBER_0]: "なし", [WaterType.NUMBER_MINUS_5]: "水辺", [WaterType.NUMBER_MINUS_10]: "水中" },
  depth: { [SpriteDepth.NUMBER_0]: "Floor - 床", [SpriteDepth.NUMBER_1]: "Rug - 床置き", [SpriteDepth.NUMBER_2]: "Taller - 高さあり", [SpriteDepth.NUMBER_3]: "Player - 最前面" },
};

const numericSelectOptions: Record<string, { value: number; label: string }[]> = Object.fromEntries(
  Object.entries(numericSelectLabels).map(([field, labels]) =>
    [field, Object.entries(labels).map(([val, label]) => {
      const v = Number(val);
      return { value: v, label: `${label} (${v >= 0 ? "+" : ""}${v})`.replace("+0", "0") };
    })],
  ),
);

// 配列要素内のフィールドで、インクリメンタル検索セレクトにするもの
// parentField は配列フィールド名、field はオブジェクト内のフィールド名
// optionsSource はどのリソースの name 一覧を使うか
type SearchableFieldDef = {
  parentField: string;
  field: string;
  optionsSource: string; // "items" | "members" | "skills"
};

const searchableFields: Record<string, SearchableFieldDef[]> = {
  professions: [
    { parentField: "items", field: "name", optionsSource: "items" },
    { parentField: "equips", field: "name", optionsSource: "items" },
    { parentField: "skills", field: "id", optionsSource: "skills" },
  ],
  recipes: [
    { parentField: "inputs", field: "name", optionsSource: "items" },
  ],
  "command-tables": [
    { parentField: "entries", field: "weapon", optionsSource: "items" },
  ],
  "drop-tables": [
    { parentField: "entries", field: "material", optionsSource: "items" },
  ],
  "item-tables": [
    { parentField: "entries", field: "itemName", optionsSource: "items" },
  ],
  "enemy-tables": [
    { parentField: "entries", field: "enemyName", optionsSource: "members" },
  ],
};

// トップレベルフィールドで、インクリメンタル検索セレクトにするもの
// フィールド名 → 参照先リソース
const searchableTopLevelFields: Record<string, Record<string, string>> = {
  members: {
    commandTableName: "command-tables",
    dropTableName: "drop-tables",
  },
};

// スキルID一覧（Go側のAllSkillIDsと同期）
const allSkillIDs = [
  "sword", "spear", "fist", "weight_bearing",
  "bow", "handgun", "rifle", "cannon", "exploration",
  "crafting", "smithing", "negotiation",
  "sprinting", "stealth", "night_vision",
  "cold_resist", "heat_resist", "hunger_resist", "healing",
  "heavy_armor", "fire_resist", "thunder_resist", "chill_resist", "photon_resist",
];

interface ResourcePageProps {
  resource: string;
  label: string;
  nameField?: string;
}

export function ResourcePage({
  resource,
  label,
  nameField = "name",
}: ResourcePageProps) {
  const { data, isLoading, error } = useResourceList<Record<string, JsonValue>>(resource);
  const updateResource = useResourceUpdate<Record<string, JsonValue>>(resource);
  const createResource = useResourceCreate<Record<string, JsonValue>>(resource);
  const deleteResource = useResourceDelete(resource);
  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
  const [editData, setEditData] = useState<Record<string, JsonValue> | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [confirmDeleteIndex, setConfirmDeleteIndex] = useState<number | null>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // 他リソースの参照用データを取得する
  // 配列要素フィールドとトップレベルフィールドの両方から必要なソースを収集する
  const neededSources = useMemo(() => {
    const sources = new Set<string>();
    searchableFields[resource]?.forEach((f) => sources.add(f.optionsSource));
    const topLevel = searchableTopLevelFields[resource];
    if (topLevel) Object.values(topLevel).forEach((s) => sources.add(s));
    return sources;
  }, [resource]);

  const itemsQuery = useResourceList<Record<string, JsonValue>>(neededSources.has("items") ? "items" : "");
  const membersQuery = useResourceList<Record<string, JsonValue>>(neededSources.has("members") ? "members" : "");
  const commandTablesQuery = useResourceList<Record<string, JsonValue>>(neededSources.has("command-tables") ? "command-tables" : "");
  const dropTablesQuery = useResourceList<Record<string, JsonValue>>(neededSources.has("drop-tables") ? "drop-tables" : "");

  // 参照先の名前リストを構築
  const referenceOptions = useMemo(() => {
    const opts: Record<string, string[]> = {};
    if (itemsQuery.data?.data) {
      opts["items"] = itemsQuery.data.data.map((item) => String(item["name"] ?? ""));
    }
    if (membersQuery.data?.data) {
      opts["members"] = membersQuery.data.data.map((m) => String(m["name"] ?? ""));
    }
    if (commandTablesQuery.data?.data) {
      opts["command-tables"] = commandTablesQuery.data.data.map((t) => String(t["name"] ?? ""));
    }
    if (dropTablesQuery.data?.data) {
      opts["drop-tables"] = dropTablesQuery.data.data.map((t) => String(t["name"] ?? ""));
    }
    opts["skills"] = allSkillIDs;
    return opts;
  }, [itemsQuery.data, membersQuery.data, commandTablesQuery.data, dropTablesQuery.data]);

  const items = data?.data ?? [];

  // 選択中のアイテムが更新されたら editData を同期する
  useEffect(() => {
    if (selectedIndex !== null && items[selectedIndex]) {
      setEditData(structuredClone(items[selectedIndex]) as Record<string, JsonValue>);
    }
  }, [data, selectedIndex]);

  if (isLoading) return <Text>読み込み中...</Text>;
  if (error) return <Text color="red.500">エラー: {String(error)}</Text>;

  function handleCreate() {
    const template: Record<string, JsonValue> = createTemplates[resource]
      ? structuredClone(createTemplates[resource]) as Record<string, JsonValue>
      : { [nameField]: "新規" };
    createResource.mutate(template, {
      // invalidateQueries 完了後に呼ばれる。レスポンスにソート後のインデックスが含まれる
      onSuccess: (result) => {
        const newIndex = (result as { index: number }).index;
        setSelectedIndex(newIndex);
        setSaveError(null);
      },
    });
  }

  function handleSelect(index: number) {
    setSelectedIndex(index);
    setEditData(structuredClone(items[index]) as Record<string, JsonValue>);
    setSaveError(null);
  }

  function handleSave() {
    if (selectedIndex === null || !editData) return;
    setSaveError(null);
    setSaveSuccess(false);
    updateResource.mutate(
      { index: selectedIndex, data: editData },
      {
        onSuccess: () => {
          setSaveSuccess(true);
          setTimeout(() => setSaveSuccess(false), 2000);
        },
        onError: (err) => setSaveError(String(err)),
      },
    );
  }

  function handleDelete(index: number) {
    if (confirmDeleteIndex === index) {
      // 2回目クリックで実行
      setConfirmDeleteIndex(null);
      deleteResource.mutate(index, {
        onSuccess: () => {
          if (selectedIndex === index) {
            setSelectedIndex(null);
            setEditData(null);
          }
        },
      });
    } else {
      // 1回目クリックで確認状態に。3秒後に自動解除
      setConfirmDeleteIndex(index);
      setTimeout(() => setConfirmDeleteIndex((prev) => prev === index ? null : prev), 3000);
    }
  }

  function handleFieldChange(path: string[], value: JsonValue) {
    if (!editData || path.length === 0) return;
    const next = structuredClone(editData) as Record<string, JsonValue>;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    let obj: any = next;
    for (let i = 0; i < path.length - 1; i++) {
      obj = obj[path[i]!];
    }
    obj[path[path.length - 1]!] = value;
    setEditData(next);
  }

  function handleToggleSection(key: string) {
    if (!editData) return;
    const next = structuredClone(editData) as Record<string, JsonValue>;
    if (next[key] != null) {
      // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
      delete next[key];
    } else {
      // デフォルト値があればそれを使う
      const defaults = optionalSections[resource];
      const template = defaults?.[key];
      next[key] = template != null ? structuredClone(template) : {};
    }
    setEditData(next);
  }

  return (
    <Flex gap="4" h="100%">
      {/* 一覧 */}
      <Box
        ref={listRef}
        w="280px"
        flexShrink={0}
        overflowY="auto"
        borderRight="1px solid"
        borderColor="border"
        pr="3"
      >
        <Flex justify="space-between" align="center" mb="3">
          <Heading size="md">
            {label}
            <Badge ml="2" colorPalette="gray">{items.length}</Badge>
          </Heading>
          <Button
            size="xs"
            variant="outline"
            onClick={handleCreate}
            loading={createResource.isPending}
          >
            ＋
          </Button>
        </Flex>
        <Stack gap="1">
          {items.map((item, index) => (
            <Flex
              key={index}
              px="2"
              py="1"
              borderRadius="md"
              cursor="pointer"
              bg={selectedIndex === index ? "bg.emphasized" : undefined}
              _hover={{ bg: "bg.muted" }}
              justify="space-between"
              align="center"
              onClick={() => handleSelect(index)}
            >
              <Text fontSize="sm" truncate>
                {String(item[nameField] ?? `#${index}`)}
              </Text>
              <Button
                size="xs"
                variant={confirmDeleteIndex === index ? "solid" : "ghost"}
                colorPalette="red"
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(index);
                }}
              >
                {confirmDeleteIndex === index ? "本当に?" : "×"}
              </Button>
            </Flex>
          ))}
        </Stack>
      </Box>

      {/* 編集エリア */}
      <Box flex="1" overflowY="auto">
        {editData ? (
          <>
            <Flex justify="space-between" align="center" mb="4" position="sticky" top="0" bg="bg" zIndex="1" py="2">
              <Heading size="md">
                {String(editData[nameField] ?? `#${selectedIndex}`)}
              </Heading>
              <Flex align="center" gap="2">
                {saveSuccess && (
                  <Text fontSize="sm" color="green.500" fontWeight="bold">保存しました</Text>
                )}
                <Button
                  size="sm"
                  colorPalette="blue"
                  onClick={handleSave}
                  loading={updateResource.isPending}
                >
                  保存
                </Button>
              </Flex>
            </Flex>
            {saveError && (
              <Text color="red.500" fontSize="sm" mb="2">{saveError}</Text>
            )}
            <FieldGroup
              data={editData}
              path={[]}
              rootData={editData}
              resource={resource}
              referenceOptions={referenceOptions}
              onChange={handleFieldChange}
              onToggleSection={handleToggleSection}
            />
            {/* オプショナルセクション追加ボタン */}
            {optionalSections[resource] && (() => {
              const missing = Object.keys(optionalSections[resource]!).filter((k) => !(k in editData));
              if (missing.length === 0) return null;
              return (
                <Flex gap="2" mt="4" wrap="wrap" align="center">
                  <Text fontSize="sm" color="fg.muted">セクション追加:</Text>
                  {missing.map((key) => (
                    <Button
                      key={key}
                      size="xs"
                      variant="outline"
                      onClick={() => handleToggleSection(key)}
                    >
                      + {key}
                    </Button>
                  ))}
                </Flex>
              );
            })()}
          </>
        ) : (
          <Text color="fg.muted">左の一覧から項目を選択してください</Text>
        )}
      </Box>
    </Flex>
  );
}

// フィールドグループ: オブジェクトのキーを再帰的にフォーム描画する
function FieldGroup({
  data,
  path,
  rootData,
  resource,
  referenceOptions,
  onChange,
  onToggleSection,
}: {
  data: Record<string, JsonValue>;
  path: string[];
  rootData: Record<string, JsonValue>;
  resource: string;
  referenceOptions: Record<string, string[]>;
  onChange: (path: string[], value: JsonValue) => void;
  onToggleSection: (key: string) => void;
}) {
  const entries = Object.entries(data);
  // プリミティブフィールドとオブジェクトフィールドに分ける
  const primitiveFields = entries.filter(([, v]) => !isObject(v) && !Array.isArray(v));
  const objectFields = entries.filter(([, v]) => isObject(v));
  const arrayFields = entries.filter(([, v]) => Array.isArray(v));

  // spriteSheetName はルートデータまたは現在のデータから取得する
  const spriteSheetName = String(
    data["spriteSheetName"] ?? rootData["spriteSheetName"] ?? "",
  );

  return (
    <Stack gap="3">
      {primitiveFields.map(([key, value]) => {
        // SearchableSelectを使うか判定する
        // 1. トップレベルフィールド: searchableTopLevelFields で定義されたもの
        // 2. 配列要素内フィールド: path例 ["items", "0"] → parentField="items"
        const topLevelSource = path.length === 0
          ? searchableTopLevelFields[resource]?.[key]
          : undefined;
        const parentField = !topLevelSource && path.length >= 2 && /^\d+$/.test(path[path.length - 1]!)
          ? path[path.length - 2]!
          : undefined;
        const searchableDef = parentField
          ? searchableFields[resource]?.find((f) => f.parentField === parentField && f.field === key)
          : undefined;
        const searchableOpts = topLevelSource
          ? referenceOptions[topLevelSource]
          : searchableDef ? referenceOptions[searchableDef.optionsSource] : undefined;

        return (
          <FieldRow
            key={key}
            label={key}
            value={value}
            spriteSheetName={key === "spriteKey" ? spriteSheetName : undefined}
            searchableOptions={searchableOpts}
            onChange={(v) => onChange([...path, key], v)}
          />
        );
      })}
      {arrayFields.map(([key, value]) => (
        <ArrayField
          key={key}
          label={key}
          items={value as JsonValue[]}
          path={[...path, key]}
          rootData={rootData}
          resource={resource}
          referenceOptions={referenceOptions}
          onChange={onChange}
        />
      ))}
      {objectFields.map(([key, value]) => (
        <Fieldset.Root key={key} borderWidth="1px" borderRadius="md" p="3">
          <Fieldset.Legend fontSize="sm" fontWeight="bold" px="1">
            {key}
            {path.length === 0 && (
              <Button
                size="xs"
                variant="ghost"
                colorPalette="red"
                ml="2"
                onClick={() => onToggleSection(key)}
              >
                削除
              </Button>
            )}
          </Fieldset.Legend>
          <FieldGroup
            data={value as Record<string, JsonValue>}
            path={[...path, key]}
            rootData={rootData}
            resource={resource}
            referenceOptions={referenceOptions}
            onChange={onChange}
            onToggleSection={onToggleSection}
          />
        </Fieldset.Root>
      ))}
    </Stack>
  );
}

// 単一フィールド
function FieldRow({
  label,
  value,
  spriteSheetName,
  searchableOptions,
  onChange,
}: {
  label: string;
  value: JsonValue;
  spriteSheetName?: string;
  searchableOptions?: string[];
  onChange: (v: JsonValue) => void;
}) {
  if (typeof value === "boolean") {
    return (
      <Flex align="center" gap="3">
        <Text fontSize="sm" w="180px" flexShrink={0} color="fg.muted">{label}</Text>
        <Switch
          checked={value}
          onCheckedChange={(e) => onChange(e.checked)}
          size="sm"
        />
      </Flex>
    );
  }

  if (typeof value === "number") {
    // 数値enum型フィールド
    const numericOpts = numericSelectOptions[label];
    if (numericOpts) {
      return (
        <Flex align="center" gap="3">
          <Text fontSize="sm" w="180px" flexShrink={0} color="fg.muted">{label}</Text>
          <NativeSelectRoot size="sm" flex="1">
            <NativeSelectField
              value={String(value)}
              onChange={(e) => onChange(parseInt(e.target.value, 10))}
            >
              {numericOpts.map((opt) => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </NativeSelectField>
          </NativeSelectRoot>
        </Flex>
      );
    }

    return (
      <Flex align="center" gap="3">
        <Text fontSize="sm" w="180px" flexShrink={0} color="fg.muted">{label}</Text>
        <Input
          size="sm"
          type="number"
          step="any"
          value={value}
          onChange={(e) => {
            const v = e.target.value;
            onChange(v.includes(".") ? parseFloat(v) : parseInt(v, 10));
          }}
        />
      </Flex>
    );
  }

  // spriteKey フィールドにはスプライト選択UIを使う
  if (label === "spriteKey" && spriteSheetName) {
    return (
      <Flex align="center" gap="3">
        <Text fontSize="sm" w="180px" flexShrink={0} color="fg.muted">{label}</Text>
        <SpriteSelect
          sheetName={spriteSheetName}
          value={String(value ?? "")}
          onChange={(key) => onChange(key)}
        />
      </Flex>
    );
  }

  // インクリメンタル検索セレクト（他リソース参照フィールド）
  if (searchableOptions) {
    return (
      <Flex align="center" gap="3">
        <Text fontSize="sm" w="180px" flexShrink={0} color="fg.muted">{label}</Text>
        <SearchableSelect
          options={searchableOptions}
          value={String(value ?? "")}
          onChange={(v) => onChange(v)}
        />
      </Flex>
    );
  }

  // 選択式フィールド
  const selectDef = selectFieldOptions[label];
  if (selectDef) {
    return (
      <Flex align="center" gap="3">
        <Text fontSize="sm" w="180px" flexShrink={0} color="fg.muted">{label}</Text>
        <NativeSelectRoot size="sm" flex="1">
          <NativeSelectField
            value={String(value ?? "")}
            onChange={(e) => onChange(e.target.value)}
          >
            {selectDef.allowEmpty && <option value="">（なし）</option>}
            {selectDef.options.map((opt) => (
              <option key={opt} value={opt}>{opt}</option>
            ))}
          </NativeSelectField>
        </NativeSelectRoot>
      </Flex>
    );
  }

  // string or null
  return (
    <Flex align="center" gap="3">
      <Text fontSize="sm" w="180px" flexShrink={0} color="fg.muted">{label}</Text>
      <Input
        size="sm"
        value={String(value ?? "")}
        onChange={(e) => onChange(e.target.value)}
      />
    </Flex>
  );
}

// 配列要素追加時のデフォルト値を決定する
function getArrayElementDefault(label: string, resource: string, items: JsonValue[]): JsonValue {
  // entriesはリソースごとに異なるテンプレートを使う
  if (label === "entries" && entriesTemplates[resource]) {
    return structuredClone(entriesTemplates[resource]!);
  }
  // 既知のフィールド名にテンプレートがある場合
  if (label in arrayElementTemplates) {
    const tmpl = arrayElementTemplates[label]!;
    // テンプレートが空オブジェクトの場合、既存要素のキー構造を使う
    if (isObject(tmpl) && Object.keys(tmpl as Record<string, JsonValue>).length === 0 && items.length > 0) {
      const sample = items[0]!;
      if (isObject(sample)) {
        const empty: Record<string, JsonValue> = {};
        for (const [k, v] of Object.entries(sample as Record<string, JsonValue>)) {
          if (typeof v === "number") empty[k] = 0;
          else if (typeof v === "boolean") empty[k] = false;
          else empty[k] = "";
        }
        return empty;
      }
    }
    return structuredClone(tmpl);
  }
  // 既存要素から型を推論
  if (items.length > 0) {
    const sample = items[0]!;
    if (typeof sample === "number") return 0;
    if (typeof sample === "string") return "";
    if (isObject(sample)) {
      const empty: Record<string, JsonValue> = {};
      for (const [k, v] of Object.entries(sample as Record<string, JsonValue>)) {
        if (typeof v === "number") empty[k] = 0;
        else if (typeof v === "boolean") empty[k] = false;
        else empty[k] = "";
      }
      return empty;
    }
  }
  return "";
}

// 配列フィールド
function ArrayField({
  label,
  items,
  path,
  rootData,
  resource,
  referenceOptions,
  onChange,
}: {
  label: string;
  items: JsonValue[];
  path: string[];
  rootData: Record<string, JsonValue>;
  resource: string;
  referenceOptions: Record<string, string[]>;
  onChange: (path: string[], value: JsonValue) => void;
}) {
  const handleAdd = () => {
    const newItem = getArrayElementDefault(label, resource, items);
    onChange(path, [...items, newItem]);
  };

  const handleRemove = (index: number) => {
    onChange(path, items.filter((_, i) => i !== index));
  };

  const isPrimitive = items.length === 0
    ? typeof getArrayElementDefault(label, resource, items) !== "object"
    : items.every((v) => typeof v === "string" || typeof v === "number");

  // プリミティブ配列
  if (isPrimitive) {
    return (
      <Flex align="start" gap="3">
        <Text fontSize="sm" w="180px" flexShrink={0} color="fg.muted" pt="1">{label}</Text>
        <Stack gap="1" flex="1">
          {items.map((item, i) => (
            <Flex key={i} gap="1">
              <Input
                size="sm"
                value={String(item)}
                onChange={(e) => {
                  const next = [...items];
                  next[i] = typeof item === "number" ? parseFloat(e.target.value) : e.target.value;
                  onChange(path, next);
                }}
              />
              <Button size="xs" variant="ghost" colorPalette="red" onClick={() => handleRemove(i)}>×</Button>
            </Flex>
          ))}
          <Button size="xs" variant="outline" onClick={handleAdd}>＋ 追加</Button>
        </Stack>
      </Flex>
    );
  }

  // オブジェクト配列
  return (
    <Fieldset.Root borderWidth="1px" borderRadius="md" p="3">
      <Fieldset.Legend fontSize="sm" fontWeight="bold" px="1">
        {label} ({items.length})
      </Fieldset.Legend>
      <Stack gap="3">
        {items.map((item, i) => (
          <Box key={i} borderWidth="1px" borderRadius="md" p="2">
            <Flex justify="space-between" align="center" mb="1">
              <Text fontSize="xs" color="fg.subtle">#{i}</Text>
              <Button size="xs" variant="ghost" colorPalette="red" onClick={() => handleRemove(i)}>×</Button>
            </Flex>
            {isObject(item) ? (
              <FieldGroup
                data={item as Record<string, JsonValue>}
                path={[...path, String(i)]}
                rootData={rootData}
                resource={resource}
                referenceOptions={referenceOptions}
                onChange={onChange}
                onToggleSection={() => {}}
              />
            ) : (
              <Text fontSize="sm">{JSON.stringify(item)}</Text>
            )}
          </Box>
        ))}
        <Button size="xs" variant="outline" onClick={handleAdd}>＋ 追加</Button>
      </Stack>
    </Fieldset.Root>
  );
}

function isObject(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === "object" && !Array.isArray(v);
}
