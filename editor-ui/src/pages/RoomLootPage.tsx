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
import type { BalanceReport } from "../generated";

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

// 部屋役割の id を日本語ラベルへ写す。無いものは id をそのまま出す。
const roleLabels: Record<string, string> = {
  all: "全体",
  other: "その他",
  main: "主室",
  genkan: "玄関",
  corridor: "廊下",
  living: "居間",
  kitchen: "台所",
  bedroom: "寝室",
  dressing: "脱衣所",
  bath: "浴室",
  toilet: "トイレ",
  storage: "物置",
  storeroom: "倉庫",
  coldroom: "冷蔵室",
  office: "事務室",
  restroom: "トイレ",
  sales: "売場",
  waiting: "待合",
  exam: "診察室",
  pharmacy: "薬局",
};

export function RoomLootPage() {
  const [data, setData] = useState<BalanceReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [facility, setFacility] = useState<string>("");
  const [role, setRole] = useState<string>("");

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
          setRole(first.rooms[0]?.role ?? "");
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

  const currentFac =
    roomLoot.find((f) => f.facility === facility) ?? roomLoot[0];
  if (!currentFac) return null;
  const currentRoom =
    currentFac.rooms.find((r) => r.role === role) ?? currentFac.rooms[0];
  if (!currentRoom) return null;

  // 期待個数の割合バー。上位を色分けし、残りは「その他」へ畳んで読みやすくする。
  const topN = 12;
  const barEntries = currentRoom.items
    .slice(0, topN)
    .map((it) => ({ name: it.name, weight: it.expectedCount }));
  const restWeight = currentRoom.items
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
        各施設を {currentFac.trials} 回生成し、床 loot
        と収納の中身を実際の抽選で materialize
        して集計したもの。確率は1棟あたり1個以上出る割合、期待個数は1棟あたりの平均。
      </Text>

      <Flex align="center" gap="4" mb="4" flexWrap="wrap">
        <Flex align="center" gap="2">
          <Text fontSize="sm">施設:</Text>
          <NativeSelect.Root size="sm" width="auto">
            <NativeSelect.Field
              value={facility}
              onChange={(e) => {
                const f = e.target.value;
                setFacility(f);
                const fac = roomLoot.find((x) => x.facility === f);
                setRole(fac?.rooms[0]?.role ?? "all");
              }}
            >
              {roomLoot.map((f) => (
                <option key={f.facility} value={f.facility}>
                  {facilityLabels[f.facility] ?? f.facility}
                </option>
              ))}
            </NativeSelect.Field>
          </NativeSelect.Root>
        </Flex>

        <Flex align="center" gap="2">
          <Text fontSize="sm">部屋:</Text>
          <NativeSelect.Root size="sm" width="auto">
            <NativeSelect.Field
              value={currentRoom.role}
              onChange={(e) => setRole(e.target.value)}
            >
              {currentFac.rooms.map((r) => (
                <option key={r.role} value={r.role}>
                  {roleLabels[r.role] ?? r.role}
                </option>
              ))}
            </NativeSelect.Field>
          </NativeSelect.Root>
        </Flex>

        <Text fontSize="sm" color="fg.muted">
          {currentRoom.items.length} 種
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
          {currentRoom.items.map((it) => (
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
