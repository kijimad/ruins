import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import axios from "axios";
import {
  Box,
  Heading,
  Text,
  Stack,
  Flex,
  Badge,
  Table,
  NativeSelect,
} from "@chakra-ui/react";
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  AreaChart,
  Area,
  BarChart,
  Bar,
  ReferenceLine,
} from "recharts";

interface DepthStat {
  depth: number;
  medianHP: number;
  p5HP: number;
  p95HP: number;
  medianHPBeforeHeal: number;
  p5HPBeforeHeal: number;
  p95HPBeforeHeal: number;
  suddenDeathRate: number;
  weaponDistribution?: Record<string, number>;
  medianHunger: number;
  p5Hunger: number;
  p95Hunger: number;
}

interface TrialDepthStat {
  depth: number;
  hp: number;
  hpBeforeHeal: number;
  weapon: string;
  hunger: number;
}

interface TrialResult {
  index: number;
  reachedDepth: number;
  died: boolean;
  depths: TrialDepthStat[];
}

interface EnemyTableRun {
  name: string;
  maxDepth: number;
  trials: number;
  medianDepth: number;
  deathRate: number;
  depths: DepthStat[];
  trialData?: TrialResult[];
}

interface PlayerInfo {
  name: string;
  hp: number;
  strength: number;
  sensation: number;
  dexterity: number;
  agility: number;
  defense: number;
}

interface WeaponInfo {
  name: string;
  damage: number;
  accuracy: number;
}

interface BalanceData {
  mode: string;
  player?: PlayerInfo;
  weapon?: WeaponInfo;
  enemyTables?: EnemyTableRun[];
}

const COLORS = [
  "#8884d8",
  "#82ca9d",
  "#ffc658",
  "#ff7300",
  "#d0ed57",
  "#a4de6c",
];

function useBalance() {
  return useQuery<BalanceData>({
    queryKey: ["balance"],
    queryFn: async () => {
      const res = await axios.get<BalanceData>("/api/v1/balance");
      return res.data;
    },
  });
}

function HPChart({ run }: { run: EnemyTableRun }) {
  const data = run.depths.map((d) => ({
    depth: d.depth,
    "戦闘後HP P95": d.p95HPBeforeHeal,
    "戦闘後HP中央値": d.medianHPBeforeHeal,
    "戦闘後HP P5": d.p5HPBeforeHeal,
    "回復後HP中央値": d.medianHP,
  }));

  return (
    <ResponsiveContainer width="100%" height={300}>
      <AreaChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="depth" label={{ value: "深度", position: "bottom" }} />
        <YAxis label={{ value: "HP", angle: -90, position: "insideLeft" }} />
        <Tooltip />
        <Legend />
        <Area
          type="monotone"
          dataKey="戦闘後HP P95"
          stroke="#82ca9d"
          fill="#82ca9d"
          fillOpacity={0.2}
        />
        <Area
          type="monotone"
          dataKey="戦闘後HP中央値"
          stroke="#8884d8"
          fill="#8884d8"
          fillOpacity={0.3}
        />
        <Area
          type="monotone"
          dataKey="戦闘後HP P5"
          stroke="#ff7300"
          fill="#ff7300"
          fillOpacity={0.2}
        />
        <Line
          type="monotone"
          dataKey="回復後HP中央値"
          stroke="#888"
          strokeDasharray="5 5"
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

function DeathRateChart({ run }: { run: EnemyTableRun }) {
  const data = run.depths.map((d) => ({
    depth: d.depth,
    "突然死率(%)": Number((d.suddenDeathRate * 100).toFixed(1)),
  }));

  return (
    <ResponsiveContainer width="100%" height={250}>
      <BarChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="depth" label={{ value: "深度", position: "bottom" }} />
        <YAxis
          label={{ value: "%", angle: -90, position: "insideLeft" }}
          domain={[0, 100]}
        />
        <Tooltip />
        <Bar dataKey="突然死率(%)" fill="#ff7300" />
      </BarChart>
    </ResponsiveContainer>
  );
}

const WEAPON_COLORS: Record<string, string> = {
  素手: "#999999",
  木刀: "#d4a574",
  鉄のナイフ: "#7799cc",
  氷の槍: "#66ccee",
};

function WeaponChart({ run }: { run: EnemyTableRun }) {
  const weaponNames = new Set<string>();
  for (const d of run.depths) {
    if (d.weaponDistribution) {
      for (const name of Object.keys(d.weaponDistribution)) {
        weaponNames.add(name);
      }
    }
  }
  if (weaponNames.size === 0) return null;

  const data = run.depths.map((d) => {
    const total = d.weaponDistribution
      ? Object.values(d.weaponDistribution).reduce((a, b) => a + b, 0)
      : 0;
    const row: Record<string, number> = { depth: d.depth };
    for (const name of weaponNames) {
      const count = d.weaponDistribution?.[name] ?? 0;
      row[name] = total > 0 ? Math.round((count / total) * 100) : 0;
    }
    return row;
  });

  const names = [...weaponNames];

  return (
    <ResponsiveContainer width="100%" height={250}>
      <BarChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="depth" label={{ value: "深度", position: "bottom" }} />
        <YAxis
          label={{ value: "%", angle: -90, position: "insideLeft" }}
          domain={[0, 100]}
        />
        <Tooltip />
        <Legend />
        {names.map((name) => (
          <Bar
            key={name}
            dataKey={name}
            stackId="weapon"
            fill={WEAPON_COLORS[name] ?? COLORS[names.indexOf(name) % COLORS.length]}
          />
        ))}
      </BarChart>
    </ResponsiveContainer>
  );
}

function HungerChart({ run }: { run: EnemyTableRun }) {
  const data = run.depths.map((d) => ({
    depth: d.depth,
    "空腹度 P95": d.p95Hunger,
    "空腹度中央値": d.medianHunger,
    "空腹度 P5": d.p5Hunger,
  }));

  return (
    <ResponsiveContainer width="100%" height={250}>
      <AreaChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="depth" label={{ value: "深度", position: "bottom" }} />
        <YAxis
          label={{ value: "空腹度", angle: -90, position: "insideLeft" }}
          domain={[0, 500]}
        />
        <Tooltip />
        <Legend />
        <ReferenceLine
          y={500 * 0.33}
          stroke="#ff7300"
          strokeDasharray="3 3"
          label="飢餓"
        />
        <ReferenceLine
          y={500 * 0.66}
          stroke="#ffc658"
          strokeDasharray="3 3"
          label="空腹"
        />
        <Area
          type="monotone"
          dataKey="空腹度 P95"
          stroke="#82ca9d"
          fill="#82ca9d"
          fillOpacity={0.2}
        />
        <Area
          type="monotone"
          dataKey="空腹度中央値"
          stroke="#8884d8"
          fill="#8884d8"
          fillOpacity={0.3}
        />
        <Area
          type="monotone"
          dataKey="空腹度 P5"
          stroke="#ff7300"
          fill="#ff7300"
          fillOpacity={0.2}
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

function TrialDetail({
  run,
  playerHP,
}: {
  run: EnemyTableRun;
  playerHP?: number;
}) {
  const [selectedIndex, setSelectedIndex] = useState(0);
  const trials = run.trialData;
  if (!trials || trials.length === 0) return null;

  const trial = trials[selectedIndex];
  if (!trial) return null;

  const chartData = trial.depths.map((d) => ({
    depth: d.depth,
    "戦闘後HP": d.hpBeforeHeal,
    "回復後HP": d.hp,
    "空腹度": d.hunger,
  }));

  return (
    <Box>
      <Flex align="center" gap="3" mb="3">
        <Heading size="sm">試行詳細</Heading>
        <NativeSelect.Root size="sm" width="320px">
          <NativeSelect.Field
            value={selectedIndex}
            onChange={(e) => setSelectedIndex(Number(e.target.value))}
          >
            {trials.map((t) => (
              <option key={t.index} value={t.index}>
                #{t.index} - 深度{t.reachedDepth}
                {t.died ? " (死亡)" : " (生存)"}
              </option>
            ))}
          </NativeSelect.Field>
        </NativeSelect.Root>
        <Badge colorPalette={trial.died ? "red" : "green"}>
          {trial.died ? "死亡" : "生存"} - 深度{trial.reachedDepth}
        </Badge>
      </Flex>

      <ResponsiveContainer width="100%" height={250}>
        <LineChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis
            dataKey="depth"
            label={{ value: "深度", position: "bottom" }}
          />
          <YAxis
            yAxisId="hp"
            label={{ value: "HP", angle: -90, position: "insideLeft" }}
          />
          <YAxis
            yAxisId="hunger"
            orientation="right"
            domain={[0, 500]}
            label={{ value: "空腹度", angle: 90, position: "insideRight" }}
          />
          <Tooltip />
          <Legend />
          {playerHP && (
            <ReferenceLine
              yAxisId="hp"
              y={playerHP}
              stroke="#ccc"
              strokeDasharray="3 3"
              label="最大HP"
            />
          )}
          <Line
            yAxisId="hp"
            type="monotone"
            dataKey="戦闘後HP"
            stroke="#ff7300"
            strokeWidth={2}
          />
          <Line
            yAxisId="hp"
            type="monotone"
            dataKey="回復後HP"
            stroke="#82ca9d"
            strokeWidth={2}
          />
          <Line
            yAxisId="hunger"
            type="monotone"
            dataKey="空腹度"
            stroke="#8884d8"
            strokeWidth={2}
            strokeDasharray="5 5"
          />
        </LineChart>
      </ResponsiveContainer>

      <Table.Root size="sm" mt="2">
        <Table.Header>
          <Table.Row>
            <Table.ColumnHeader>深度</Table.ColumnHeader>
            <Table.ColumnHeader>戦闘後HP</Table.ColumnHeader>
            <Table.ColumnHeader>回復後HP</Table.ColumnHeader>
            <Table.ColumnHeader>武器</Table.ColumnHeader>
            <Table.ColumnHeader>空腹度</Table.ColumnHeader>
          </Table.Row>
        </Table.Header>
        <Table.Body>
          {trial.depths.map((d) => (
            <Table.Row key={d.depth}>
              <Table.Cell>{d.depth}</Table.Cell>
              <Table.Cell>{d.hpBeforeHeal}</Table.Cell>
              <Table.Cell>{d.hp}</Table.Cell>
              <Table.Cell>{d.weapon}</Table.Cell>
              <Table.Cell>{d.hunger}</Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>
    </Box>
  );
}

function ComparisonChart({ tables }: { tables: EnemyTableRun[] }) {
  const maxDepth = Math.max(...tables.map((t) => t.depths.length));
  const data = Array.from({ length: maxDepth }, (_, i) => {
    const row: Record<string, number> = { depth: i + 1 };
    for (const t of tables) {
      const d = t.depths.find((d) => d.depth === i + 1);
      row[t.name] = d?.medianHPBeforeHeal ?? 0;
    }
    return row;
  });

  return (
    <ResponsiveContainer width="100%" height={350}>
      <LineChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="depth" label={{ value: "深度", position: "bottom" }} />
        <YAxis label={{ value: "戦闘後HP中央値", angle: -90, position: "insideLeft" }} />
        <Tooltip />
        <Legend />
        {tables.map((t, i) => (
          <Line
            key={t.name}
            type="monotone"
            dataKey={t.name}
            stroke={COLORS[i % COLORS.length]}
            strokeWidth={2}
          />
        ))}
      </LineChart>
    </ResponsiveContainer>
  );
}

function EnemyTableSection({
  tables,
  playerHP,
}: {
  tables: EnemyTableRun[];
  playerHP?: number;
}) {
  return (
    <>
      {tables.length > 1 && (
        <Box>
          <Heading size="md" mb="3">
            テーブル間HP比較
          </Heading>
          <ComparisonChart tables={tables} />
        </Box>
      )}

      {tables.map((run) => (
        <Box key={run.name} borderWidth="1px" borderRadius="md" p="4">
          <Flex align="center" gap="3" mb="3">
            <Heading size="md">{run.name}</Heading>
            <Badge colorPalette={run.deathRate > 0.5 ? "red" : "green"}>
              死亡率 {(run.deathRate * 100).toFixed(1)}%
            </Badge>
            <Badge>到達深度中央値 {run.medianDepth}</Badge>
            <Badge>{run.trials}回試行</Badge>
          </Flex>

          <Stack gap="4">
            <Box>
              <Heading size="sm" mb="2">
                HP推移 (P5 / 中央値 / P95)
              </Heading>
              <HPChart run={run} />
            </Box>

            <Box>
              <Heading size="sm" mb="2">
                深度別突然死率
              </Heading>
              <DeathRateChart run={run} />
            </Box>

            <Box>
              <Heading size="sm" mb="2">
                武器分布
              </Heading>
              <WeaponChart run={run} />
            </Box>

            <Box>
              <Heading size="sm" mb="2">
                空腹度推移 (P5 / 中央値 / P95)
              </Heading>
              <HungerChart run={run} />
            </Box>

            <Table.Root size="sm">
              <Table.Header>
                <Table.Row>
                  <Table.ColumnHeader>深度</Table.ColumnHeader>
                  <Table.ColumnHeader>戦闘後HP中央値</Table.ColumnHeader>
                  <Table.ColumnHeader>戦闘後HP P5</Table.ColumnHeader>
                  <Table.ColumnHeader>戦闘後HP P95</Table.ColumnHeader>
                  <Table.ColumnHeader>回復後HP中央値</Table.ColumnHeader>
                  <Table.ColumnHeader>突然死率</Table.ColumnHeader>
                  <Table.ColumnHeader>空腹度</Table.ColumnHeader>
                  <Table.ColumnHeader>主要武器</Table.ColumnHeader>
                </Table.Row>
              </Table.Header>
              <Table.Body>
                {run.depths.map((d) => {
                  const topWeapon = d.weaponDistribution
                    ? Object.entries(d.weaponDistribution).sort(
                        ([, a], [, b]) => b - a,
                      )[0]
                    : null;
                  return (
                    <Table.Row key={d.depth}>
                      <Table.Cell>{d.depth}</Table.Cell>
                      <Table.Cell>{d.medianHPBeforeHeal}</Table.Cell>
                      <Table.Cell>{d.p5HPBeforeHeal}</Table.Cell>
                      <Table.Cell>{d.p95HPBeforeHeal}</Table.Cell>
                      <Table.Cell>{d.medianHP}</Table.Cell>
                      <Table.Cell>
                        {(d.suddenDeathRate * 100).toFixed(1)}%
                      </Table.Cell>
                      <Table.Cell>{d.medianHunger}</Table.Cell>
                      <Table.Cell>
                        {topWeapon ? topWeapon[0] : "-"}
                      </Table.Cell>
                    </Table.Row>
                  );
                })}
              </Table.Body>
            </Table.Root>

            <TrialDetail run={run} playerHP={playerHP} />
          </Stack>
        </Box>
      ))}
    </>
  );
}

function PlayerInfoCard({ player }: { player: PlayerInfo }) {
  return (
    <Box borderWidth="1px" borderRadius="md" p="4">
      <Heading size="sm" mb="2">
        プレイヤー: {player.name}
      </Heading>
      <Table.Root size="sm">
        <Table.Body>
          {[
            ["HP", player.hp],
            ["筋力", player.strength],
            ["感覚", player.sensation],
            ["器用", player.dexterity],
            ["敏捷", player.agility],
            ["防御", player.defense],
          ].map(([label, value]) => (
            <Table.Row key={label}>
              <Table.Cell fontWeight="bold">{label}</Table.Cell>
              <Table.Cell>{value}</Table.Cell>
            </Table.Row>
          ))}
        </Table.Body>
      </Table.Root>
    </Box>
  );
}

function WeaponInfoCard({ weapon }: { weapon: WeaponInfo }) {
  return (
    <Box borderWidth="1px" borderRadius="md" p="4">
      <Heading size="sm" mb="2">
        武器: {weapon.name}
      </Heading>
      <Table.Root size="sm">
        <Table.Body>
          <Table.Row>
            <Table.Cell fontWeight="bold">ダメージ</Table.Cell>
            <Table.Cell>{weapon.damage}</Table.Cell>
          </Table.Row>
          <Table.Row>
            <Table.Cell fontWeight="bold">命中率</Table.Cell>
            <Table.Cell>{weapon.accuracy}</Table.Cell>
          </Table.Row>
        </Table.Body>
      </Table.Root>
    </Box>
  );
}

export function BalancePage() {
  const { data, isLoading, error } = useBalance();

  if (isLoading) return <Text>読み込み中...</Text>;
  if (error)
    return (
      <Text color="fg.error">
        エラー: balance.json が見つかりません。`go run . simulate-balance`
        を実行してください。
      </Text>
    );
  if (!data) return null;

  return (
    <Stack gap="6">
      <Heading size="lg">バランスシミュレーション</Heading>

      <Stack gap="6">
        <Flex gap="6" wrap="wrap">
          {data.player && <PlayerInfoCard player={data.player} />}
          {data.weapon && <WeaponInfoCard weapon={data.weapon} />}
        </Flex>

        {data.enemyTables && (
          <EnemyTableSection
            tables={data.enemyTables}
            playerHP={data.player?.hp}
          />
        )}
      </Stack>
    </Stack>
  );
}
