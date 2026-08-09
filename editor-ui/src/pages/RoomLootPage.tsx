import { useEffect, useState } from "react";
import {
  Box,
  Heading,
  Table,
  Text,
  Flex,
  NativeSelect,
} from "@chakra-ui/react";
import { WeightBar } from "../components/WeightBar";

interface LootItemStat {
  name: string;
  prob: number;
  expectedCount: number;
  value: number;
}

interface FacilityLoot {
  facility: string;
  trials: number;
  items: LootItemStat[];
}

interface BalanceReport {
  roomLoot?: FacilityLoot[];
}

// 施設 id を日本語ラベルへ写す。balance レポートの facility 文字列と揃える。
const facilityLabels: Record<string, string> = {
  house: "民家",
  store: "商店",
  antique: "骨董品店",
  clinic: "診療所",
  lab: "研究施設",
  office: "事務所",
  depot: "倉庫",
};

export function RoomLootPage() {
  const [data, setData] = useState<BalanceReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [facility, setFacility] = useState<string>("");

  useEffect(() => {
    fetch("/api/v1/balance")
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json();
      })
      .then((d: BalanceReport) => {
        setData(d);
        const first = d.roomLoot?.[0];
        if (first) {
          setFacility(first.facility);
        }
      })
      .catch((e) => setError(e.message));
  }, []);

  if (error) return <Text color="red.500">エラー: {error}</Text>;
  if (!data) return <Text>読み込み中...</Text>;

  const roomLoot = data.roomLoot ?? [];
  if (roomLoot.length === 0) {
    return (
      <Text>
        部屋別 loot がありません。`go run . simulate-balance` で balance.json を
        再生成してください。
      </Text>
    );
  }

  const current = roomLoot.find((f) => f.facility === facility) ?? roomLoot[0];
  if (!current) return null;

  // 期待個数の割合バー。上位を色分けし、残りは「その他」へ畳んで読みやすくする。
  const topN = 12;
  const barEntries = current.items
    .slice(0, topN)
    .map((it) => ({ name: it.name, weight: it.expectedCount }));
  const restWeight = current.items
    .slice(topN)
    .reduce((s, it) => s + it.expectedCount, 0);
  if (restWeight > 0) {
    barEntries.push({ name: "その他", weight: restWeight });
  }
  const barTotal = barEntries.reduce((s, e) => s + e.weight, 0);

  return (
    <Box>
      <Heading size="lg" mb="2">
        部屋別 loot 確率
      </Heading>
      <Text color="fg.muted" mb="4" fontSize="sm">
        各施設を {current.trials} 回生成し、床 loot と収納の中身を実際の抽選で
        materialize
        して集計したもの。確率は1棟あたり1個以上出る割合、期待個数は1棟
        あたりの平均。
      </Text>

      <Flex align="center" gap="2" mb="4">
        <Text fontSize="sm">施設:</Text>
        <NativeSelect.Root size="sm" width="auto">
          <NativeSelect.Field
            value={facility}
            onChange={(e) => setFacility(e.target.value)}
          >
            {roomLoot.map((f) => (
              <option key={f.facility} value={f.facility}>
                {facilityLabels[f.facility] ?? f.facility}
              </option>
            ))}
          </NativeSelect.Field>
        </NativeSelect.Root>
        <Text fontSize="sm" color="fg.muted">
          {current.items.length} 種
        </Text>
      </Flex>

      <Box mb="4">
        <Text fontSize="sm" mb="1">
          期待個数の割合
        </Text>
        <WeightBar entries={barEntries} totalWeight={barTotal} />
      </Box>

      <Table.Root size="sm">
        <Table.Header>
          <Table.Row>
            <Table.ColumnHeader>アイテム</Table.ColumnHeader>
            <Table.ColumnHeader textAlign="right">確率</Table.ColumnHeader>
            <Table.ColumnHeader textAlign="right">期待個数</Table.ColumnHeader>
            <Table.ColumnHeader textAlign="right">価値</Table.ColumnHeader>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {current.items.map((it) => (
            <Table.Row key={it.name}>
              <Table.Cell>{it.name}</Table.Cell>
              <Table.Cell textAlign="right">
                {(it.prob * 100).toFixed(0)}%
              </Table.Cell>
              <Table.Cell textAlign="right">
                {it.expectedCount.toFixed(2)}
              </Table.Cell>
              <Table.Cell textAlign="right">{it.value}</Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>
    </Box>
  );
}
