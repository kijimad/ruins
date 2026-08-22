import { useMemo, useState } from "react";
import {
  Box,
  Heading,
  Table,
  Text,
  Flex,
  NativeSelect,
  Badge,
} from "@chakra-ui/react";
import { useResourceList } from "../hooks/useResource";
import { WeightBar } from "../components/WeightBar";

type TableType = "enemy-tables" | "item-tables" | "drop-tables";

const tableTypeLabels: Record<TableType, string> = {
  "enemy-tables": "敵テーブル",
  "item-tables": "アイテムテーブル",
  "drop-tables": "ドロップテーブル",
};

interface DangerEntry {
  name: string;
  weight: number;
}

interface EnemyTableEntry {
  enemyName: string;
  minDanger: number;
  maxDanger: number;
  weight: number;
}

interface ItemTableEntry {
  itemName: string;
  minDanger: number;
  maxDanger: number;
  weight: number;
}

interface DropTableEntry {
  material: string;
  weight: number;
}

interface TableData {
  name: string;
  entries?: (EnemyTableEntry | ItemTableEntry | DropTableEntry)[];
  [key: string]: unknown;
}

function getEntryName(
  entry: EnemyTableEntry | ItemTableEntry | DropTableEntry,
): string {
  if ("enemyName" in entry) return entry.enemyName;
  if ("itemName" in entry) return entry.itemName;
  if ("material" in entry) return entry.material;
  return "";
}

function hasDangerRange(
  entry: EnemyTableEntry | ItemTableEntry | DropTableEntry,
): entry is EnemyTableEntry | ItemTableEntry {
  return "minDanger" in entry && "maxDanger" in entry;
}

export function TableViewerPage() {
  const [tableType, setTableType] = useState<TableType>("enemy-tables");

  const enemyQuery = useResourceList<TableData>("enemy-tables");
  const itemQuery = useResourceList<TableData>("item-tables");
  const dropQuery = useResourceList<TableData>("drop-tables");

  const queryMap: Record<TableType, typeof enemyQuery> = {
    "enemy-tables": enemyQuery,
    "item-tables": itemQuery,
    "drop-tables": dropQuery,
  };

  const currentQuery = queryMap[tableType];
  const tables = currentQuery.data?.data ?? [];
  const tableNames = tables.map((t) => t.name);

  const [selectedTable, setSelectedTable] = useState<string>("");
  const activeTableName = selectedTable || tableNames[0] || "";
  const activeTable = tables.find((t) => t.name === activeTableName);
  const entries = useMemo(() => activeTable?.entries ?? [], [activeTable]);

  // 危険度ごとのエントリを構築する
  const { dangerMap, maxDanger } = useMemo(() => {
    const isDropTable = tableType === "drop-tables";
    if (isDropTable) {
      // ドロップテーブルは危険度なし。全エントリを危険度0に入れる
      const map = new Map<number, DangerEntry[]>();
      const items: DangerEntry[] = entries.map((e) => ({
        name: getEntryName(e),
        weight: e.weight,
      }));
      if (items.length > 0) map.set(0, items);
      return { dangerMap: map, maxDanger: 0 };
    }

    let max = 0;
    for (const e of entries) {
      if (hasDangerRange(e) && e.maxDanger > max) max = e.maxDanger;
    }

    const map = new Map<number, DangerEntry[]>();
    for (let d = 1; d <= max; d++) {
      const active: DangerEntry[] = [];
      for (const e of entries) {
        if (hasDangerRange(e) && d >= e.minDanger && d <= e.maxDanger) {
          active.push({ name: getEntryName(e), weight: e.weight });
        }
      }
      if (active.length > 0) {
        active.sort((a, b) => b.weight - a.weight);
        map.set(d, active);
      }
    }
    return { dangerMap: map, maxDanger: max };
  }, [entries, tableType]);

  const isLoading =
    enemyQuery.isLoading || itemQuery.isLoading || dropQuery.isLoading;
  const error = enemyQuery.error || itemQuery.error || dropQuery.error;

  if (error) return <Text color="red.500">エラー: {String(error)}</Text>;
  if (isLoading) return <Text>読み込み中...</Text>;

  return (
    <Box>
      <Heading size="lg" mb="4">
        スポーンテーブル
      </Heading>

      <Flex gap="4" mb="4" align="center">
        <Flex align="center" gap="2">
          <Text fontSize="sm" whiteSpace="nowrap">
            種別:
          </Text>
          <NativeSelect.Root size="sm" width="auto">
            <NativeSelect.Field
              value={tableType}
              onChange={(e) => {
                setTableType(e.target.value as TableType);
                setSelectedTable("");
              }}
            >
              {(Object.keys(tableTypeLabels) as TableType[]).map((key) => (
                <option key={key} value={key}>
                  {tableTypeLabels[key]}
                </option>
              ))}
            </NativeSelect.Field>
          </NativeSelect.Root>
        </Flex>

        {tableNames.length > 0 && (
          <Flex align="center" gap="2">
            <Text fontSize="sm" whiteSpace="nowrap">
              テーブル:
            </Text>
            <NativeSelect.Root size="sm" width="auto">
              <NativeSelect.Field
                value={activeTableName}
                onChange={(e) => setSelectedTable(e.target.value)}
              >
                {tableNames.map((name) => (
                  <option key={name} value={name}>
                    {name}
                  </option>
                ))}
              </NativeSelect.Field>
            </NativeSelect.Root>
          </Flex>
        )}
      </Flex>

      {entries.length === 0 ? (
        <Text color="fg.muted">エントリがありません</Text>
      ) : tableType === "drop-tables" ? (
        <DropTableView entries={dangerMap.get(0) ?? []} />
      ) : (
        <DangerTableView dangerMap={dangerMap} maxDanger={maxDanger} />
      )}
    </Box>
  );
}

function DropTableView({ entries }: { entries: DangerEntry[] }) {
  const sorted = [...entries].sort((a, b) => b.weight - a.weight);
  const totalWeight = sorted.reduce((s, e) => s + e.weight, 0);

  return (
    <Box>
      <WeightBar entries={sorted} totalWeight={totalWeight} />
      <Table.Root size="sm" mt="3">
        <Table.Header>
          <Table.Row>
            <Table.ColumnHeader>素材</Table.ColumnHeader>
            <Table.ColumnHeader textAlign="right">重み</Table.ColumnHeader>
            <Table.ColumnHeader textAlign="right">確率</Table.ColumnHeader>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {sorted.map((e) => (
            <Table.Row key={e.name}>
              <Table.Cell>{e.name}</Table.Cell>
              <Table.Cell textAlign="right">{e.weight}</Table.Cell>
              <Table.Cell textAlign="right">
                {totalWeight > 0
                  ? ((e.weight / totalWeight) * 100).toFixed(1) + "%"
                  : "-"}
              </Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>
    </Box>
  );
}

function DangerTableView({
  dangerMap,
  maxDanger,
}: {
  dangerMap: Map<number, DangerEntry[]>;
  maxDanger: number;
}) {
  return (
    <Box>
      {Array.from({ length: maxDanger }, (_, i) => i + 1).map((danger) => {
        const entries = dangerMap.get(danger);
        if (!entries) return null;
        const totalWeight = entries.reduce((s, e) => s + e.weight, 0);

        return (
          <Box key={danger} mb="4">
            <Flex align="center" gap="2" mb="1">
              <Text fontSize="sm" fontWeight="bold" whiteSpace="nowrap">
                危険度{danger}
              </Text>
              <Badge colorPalette="gray" size="sm">
                {entries.length}種
              </Badge>
            </Flex>
            <WeightBar entries={entries} totalWeight={totalWeight} />
          </Box>
        );
      })}
    </Box>
  );
}
