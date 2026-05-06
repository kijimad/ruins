import { useEffect, useRef, useState } from "react";
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
} from "@chakra-ui/react";
import { Switch } from "../components/Switch";
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
};

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
  const listRef = useRef<HTMLDivElement>(null);

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
    const template: Record<string, JsonValue> = { [nameField]: "新規" };
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
    updateResource.mutate(
      { index: selectedIndex, data: editData },
      {
        onError: (err) => setSaveError(String(err)),
      },
    );
  }

  function handleDelete(index: number) {
    const item = items[index] as Record<string, unknown> | undefined;
    const name = String(item?.[nameField] ?? `#${index}`);
    if (!confirm(`「${name}」を削除しますか?`)) return;
    deleteResource.mutate(index, {
      onSuccess: () => {
        if (selectedIndex === index) {
          setSelectedIndex(null);
          setEditData(null);
        }
      },
    });
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
                variant="ghost"
                colorPalette="red"
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(index);
                }}
              >
                ×
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
              <Button
                size="sm"
                colorPalette="blue"
                onClick={handleSave}
                loading={updateResource.isPending}
              >
                保存
              </Button>
            </Flex>
            {saveError && (
              <Text color="red.500" fontSize="sm" mb="2">{saveError}</Text>
            )}
            <FieldGroup
              data={editData}
              path={[]}
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
  onChange,
  onToggleSection,
}: {
  data: Record<string, JsonValue>;
  path: string[];
  onChange: (path: string[], value: JsonValue) => void;
  onToggleSection: (key: string) => void;
}) {
  const entries = Object.entries(data);
  // プリミティブフィールドとオブジェクトフィールドに分ける
  const primitiveFields = entries.filter(([, v]) => !isObject(v) && !Array.isArray(v));
  const objectFields = entries.filter(([, v]) => isObject(v));
  const arrayFields = entries.filter(([, v]) => Array.isArray(v));

  return (
    <Stack gap="3">
      {primitiveFields.map(([key, value]) => (
        <FieldRow
          key={key}
          label={key}
          value={value}
          onChange={(v) => onChange([...path, key], v)}
        />
      ))}
      {arrayFields.map(([key, value]) => (
        <ArrayField
          key={key}
          label={key}
          items={value as JsonValue[]}
          path={[...path, key]}
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
  onChange,
}: {
  label: string;
  value: JsonValue;
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

// 配列フィールド
function ArrayField({
  label,
  items,
  path,
  onChange,
}: {
  label: string;
  items: JsonValue[];
  path: string[];
  onChange: (path: string[], value: JsonValue) => void;
}) {
  if (items.length === 0) {
    return (
      <Flex align="center" gap="3">
        <Text fontSize="sm" w="180px" flexShrink={0} color="fg.muted">{label}</Text>
        <Text fontSize="sm" color="fg.subtle">（空）</Text>
      </Flex>
    );
  }

  // プリミティブ配列
  if (items.every((v) => typeof v === "string" || typeof v === "number")) {
    return (
      <Flex align="start" gap="3">
        <Text fontSize="sm" w="180px" flexShrink={0} color="fg.muted" pt="1">{label}</Text>
        <Stack gap="1" flex="1">
          {items.map((item, i) => (
            <Input
              key={i}
              size="sm"
              value={String(item)}
              onChange={(e) => {
                const next = [...items];
                next[i] = typeof item === "number" ? parseFloat(e.target.value) : e.target.value;
                onChange(path, next);
              }}
            />
          ))}
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
            <Text fontSize="xs" color="fg.subtle" mb="1">#{i}</Text>
            {isObject(item) ? (
              <FieldGroup
                data={item as Record<string, JsonValue>}
                path={[...path, String(i)]}
                onChange={onChange}
                onToggleSection={() => {}}
              />
            ) : (
              <Text fontSize="sm">{JSON.stringify(item)}</Text>
            )}
          </Box>
        ))}
      </Stack>
    </Fieldset.Root>
  );
}

function isObject(v: unknown): v is Record<string, unknown> {
  return v !== null && typeof v === "object" && !Array.isArray(v);
}
