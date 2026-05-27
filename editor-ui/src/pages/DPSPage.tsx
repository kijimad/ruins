import { useEffect, useState } from "react";
import {
  Box,
  Heading,
  Table,
  Text,
  Flex,
  NativeSelect,
} from "@chakra-ui/react";
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

interface BattleMetric {
  player: string;
  weapon: string;
  enemy: string;
  dps: number;
  isRanged: boolean;
}

interface BalanceReport {
  battleMetrics?: BattleMetric[];
}

type WeaponRange = "all" | "melee" | "ranged";

export function DPSPage() {
  const [data, setData] = useState<BalanceReport | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [weaponRange, setWeaponRange] = useState<WeaponRange>("all");

  useEffect(() => {
    fetch("/api/v1/balance")
      .then((r) => {
        if (!r.ok) throw new Error(`HTTP ${r.status}`);
        return r.json();
      })
      .then(setData)
      .catch((e) => setError(e.message));
  }, []);

  if (error) return <Text color="red.500">エラー: {error}</Text>;
  if (!data) return <Text>読み込み中...</Text>;

  const metrics = data.battleMetrics ?? [];
  if (metrics.length === 0) {
    return (
      <Text>
        戦闘メトリクスがありません。balance.json を再生成してください。
      </Text>
    );
  }

  // 射程フィルタを適用する
  const filtered =
    weaponRange === "all"
      ? metrics
      : metrics.filter((m) =>
          weaponRange === "ranged" ? m.isRanged : !m.isRanged,
        );

  // 武器ごとに全敵への平均DPSを集約する
  const byWeapon = new Map<string, number[]>();
  for (const m of filtered) {
    const arr = byWeapon.get(m.weapon) ?? [];
    arr.push(m.dps);
    byWeapon.set(m.weapon, arr);
  }

  const weaponDPS = [...byWeapon.entries()]
    .map(([weapon, values]) => ({
      weapon,
      dps: values.reduce((a, b) => a + b, 0) / values.length,
    }))
    .sort((a, b) => b.dps - a.dps);

  const chartData = weaponDPS.map((w) => ({
    name: w.weapon,
    DPS: Math.round(w.dps * 100) / 100,
  }));

  return (
    <Box>
      <Heading size="lg" mb="4">
        DPS メトリクス
      </Heading>

      <Flex gap="4" mb="4" align="center">
        <Flex align="center" gap="2">
          <Text fontSize="sm" whiteSpace="nowrap">
            射程:
          </Text>
          <NativeSelect.Root size="sm" width="auto">
            <NativeSelect.Field
              value={weaponRange}
              onChange={(e) => setWeaponRange(e.target.value as WeaponRange)}
            >
              <option value="all">すべて</option>
              <option value="melee">近距離</option>
              <option value="ranged">遠距離</option>
            </NativeSelect.Field>
          </NativeSelect.Root>
        </Flex>
      </Flex>

      <Box mb="6" overflowY="auto" maxH="600px">
        <Box h={`${Math.max(300, chartData.length * 32)}px`}>
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={chartData} layout="vertical">
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis type="number" />
              <YAxis
                dataKey="name"
                type="category"
                width={150}
                fontSize={12}
                interval={0}
              />
              <Tooltip />
              <Bar dataKey="DPS" fill="#4299e1" />
            </BarChart>
          </ResponsiveContainer>
        </Box>
      </Box>

      <Table.Root size="sm">
        <Table.Header>
          <Table.Row>
            <Table.ColumnHeader>武器</Table.ColumnHeader>
            <Table.ColumnHeader textAlign="right">平均DPS</Table.ColumnHeader>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {weaponDPS.map((w) => (
            <Table.Row key={w.weapon}>
              <Table.Cell>{w.weapon}</Table.Cell>
              <Table.Cell textAlign="right">{w.dps.toFixed(2)}</Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>
    </Box>
  );
}
