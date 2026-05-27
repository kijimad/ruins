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
  medianWeaponDamage: number;
  p5WeaponDamage: number;
  p95WeaponDamage: number;
  medianKillTurns: number;
  p5KillTurns: number;
  p95KillTurns: number;
  medianHunger: number;
  p5Hunger: number;
  p95Hunger: number;
  medianDamage: number;
  medianHealing: number;
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

function ChartDescription({
  model,
  criteria,
}: {
  model: string;
  criteria: string;
}) {
  return (
    <Box
      bg="bg.muted"
      borderRadius="md"
      px="3"
      py="2"
      mb="2"
      fontSize="xs"
      color="fg.muted"
    >
      <Text>
        <Text as="span" fontWeight="bold">
          モデル:
        </Text>{" "}
        {model}
      </Text>
      <Text>
        <Text as="span" fontWeight="bold">
          判断:
        </Text>{" "}
        {criteria}
      </Text>
    </Box>
  );
}

function useBalance() {
  return useQuery<BalanceData>({
    queryKey: ["balance"],
    queryFn: async () => {
      const res = await axios.get<BalanceData>("/api/v1/balance");
      return res.data;
    },
  });
}

function ResourceFlowChart({ run }: { run: EnemyTableRun }) {
  const data = run.depths.map((d) => ({
    depth: d.depth,
    被ダメージ: d.medianDamage,
    回復量: d.medianHealing,
    純消耗: d.medianDamage - d.medianHealing,
  }));

  return (
    <ResponsiveContainer width="100%" height={250}>
      <BarChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="depth" label={{ value: "深度", position: "bottom" }} />
        <YAxis label={{ value: "HP", angle: -90, position: "insideLeft" }} />
        <Tooltip />
        <Legend />
        <ReferenceLine y={0} stroke="#666" />
        <Bar dataKey="被ダメージ" fill="#ff7300" />
        <Bar dataKey="回復量" fill="#82ca9d" />
        <Bar dataKey="純消耗" fill="#8884d8" />
      </BarChart>
    </ResponsiveContainer>
  );
}

function KillTurnsChart({ run }: { run: EnemyTableRun }) {
  const data = run.depths.map((d) => ({
    depth: d.depth,
    "キルターン P95": d.p95KillTurns,
    キルターン中央値: d.medianKillTurns,
    "キルターン P5": d.p5KillTurns,
  }));

  return (
    <ResponsiveContainer width="100%" height={250}>
      <AreaChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="depth" label={{ value: "深度", position: "bottom" }} />
        <YAxis
          label={{ value: "ターン", angle: -90, position: "insideLeft" }}
        />
        <Tooltip />
        <Legend />
        <ReferenceLine
          y={5}
          stroke="#ffc658"
          strokeDasharray="3 3"
          label="目安上限"
        />
        <Area
          type="monotone"
          dataKey="キルターン P95"
          stroke="#82ca9d"
          fill="#82ca9d"
          fillOpacity={0.2}
        />
        <Area
          type="monotone"
          dataKey="キルターン中央値"
          stroke="#8884d8"
          fill="#8884d8"
          fillOpacity={0.3}
        />
        <Area
          type="monotone"
          dataKey="キルターン P5"
          stroke="#ff7300"
          fill="#ff7300"
          fillOpacity={0.2}
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

function HPChart({ run }: { run: EnemyTableRun }) {
  const data = run.depths.map((d) => ({
    depth: d.depth,
    "戦闘後HP P95": d.p95HPBeforeHeal,
    戦闘後HP中央値: d.medianHPBeforeHeal,
    "戦闘後HP P5": d.p5HPBeforeHeal,
    回復後HP中央値: d.medianHP,
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

function WeaponDamageChart({ run }: { run: EnemyTableRun }) {
  const data = run.depths.map((d) => ({
    depth: d.depth,
    "武器ダメージ P95": d.p95WeaponDamage,
    武器ダメージ中央値: d.medianWeaponDamage,
    "武器ダメージ P5": d.p5WeaponDamage,
  }));

  return (
    <ResponsiveContainer width="100%" height={250}>
      <AreaChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="depth" label={{ value: "深度", position: "bottom" }} />
        <YAxis
          label={{ value: "ダメージ", angle: -90, position: "insideLeft" }}
        />
        <Tooltip />
        <Legend />
        <Area
          type="monotone"
          dataKey="武器ダメージ P95"
          stroke="#82ca9d"
          fill="#82ca9d"
          fillOpacity={0.2}
        />
        <Area
          type="monotone"
          dataKey="武器ダメージ中央値"
          stroke="#8884d8"
          fill="#8884d8"
          fillOpacity={0.3}
        />
        <Area
          type="monotone"
          dataKey="武器ダメージ P5"
          stroke="#ff7300"
          fill="#ff7300"
          fillOpacity={0.2}
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

function HungerChart({ run }: { run: EnemyTableRun }) {
  const data = run.depths.map((d) => ({
    depth: d.depth,
    "空腹度 P95": d.p95Hunger,
    空腹度中央値: d.medianHunger,
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
    戦闘後HP: d.hpBeforeHeal,
    回復後HP: d.hp,
    空腹度: d.hunger,
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
        <YAxis
          label={{
            value: "戦闘後HP中央値",
            angle: -90,
            position: "insideLeft",
          }}
        />
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

interface ChartSection {
  title: string;
  model: string;
  criteria: string;
  render: (run: EnemyTableRun) => React.ReactNode;
}

const chartSections: ChartSection[] = [
  {
    title: "HP経済: フロアあたり消耗/回復バランス",
    model: "HP収支 = 回復量中央値 - 被ダメージ中央値",
    criteria:
      "純消耗が正の深度がゲームの壁。持続的に正なら回復ドロップを増やすか敵を弱くする",
    render: (run) => <ResourceFlowChart run={run} />,
  },
  {
    title: "戦闘力: 期待キルターン (P5 / 中央値 / P95)",
    model: "キルターン = 1戦闘あたりの平均ターン数",
    criteria:
      "3-5ターンが快適な目安。長すぎると被ダメが増大し、短すぎると緊張感がない",
    render: (run) => <KillTurnsChart run={run} />,
  },
  {
    title: "耐久: HP推移 (P5 / 中央値 / P95)",
    model: "各深度終了時の残HP分布。回復前と回復後の差がアイテムの効果量",
    criteria:
      "P5が0に近づく深度で突然死リスクが高まる。回復後HPが右肩下がりなら経済破綻",
    render: (run) => <HPChart run={run} />,
  },
  {
    title: "突然死率",
    model: "その深度に到達したランのうち、その深度で死亡した割合",
    criteria:
      "5%超は危険信号。特定深度に集中していたら、その深度の敵テーブルを確認する",
    render: (run) => <DeathRateChart run={run} />,
  },
  {
    title: "武器入手: 武器ダメージ推移 (P5 / 中央値 / P95)",
    model: "深度ごとの装備武器ダメージ値。ドロップで強い武器を拾うと上昇する",
    criteria:
      "武器成長が鈍化する区間で敵が強くなると壁になる。キルターンと合わせて確認する",
    render: (run) => <WeaponDamageChart run={run} />,
  },
  {
    title: "空腹経済: 空腹度推移 (P5 / 中央値 / P95)",
    model:
      "空腹度 = 歩行で減少、食料ドロップで回復。飢餓ラインを下回るとペナルティ",
    criteria:
      "P5が飢餓ラインを頻繁に下回るなら食料ドロップを増やすか栄養値を上げる",
    render: (run) => <HungerChart run={run} />,
  },
];

function EnemyTableSection({
  tables,
  playerHP,
}: {
  tables: EnemyTableRun[];
  playerHP?: number;
}) {
  const [selectedTable, setSelectedTable] = useState(0);
  const run = tables[selectedTable];
  if (!run) return null;

  return (
    <Stack gap="4">
      {tables.length > 1 && (
        <Box borderWidth="1px" borderRadius="md" p="4">
          <Heading size="sm" mb="1">
            テーブル間HP比較
          </Heading>
          <ComparisonChart tables={tables} />
        </Box>
      )}

      <Box borderWidth="1px" borderRadius="md" p="4">
        <Flex align="center" gap="3" mb="3" wrap="wrap">
          <NativeSelect.Root size="sm" width="auto">
            <NativeSelect.Field
              value={selectedTable}
              onChange={(e) => setSelectedTable(Number(e.target.value))}
            >
              {tables.map((t, i) => (
                <option key={t.name} value={i}>
                  {t.name}
                </option>
              ))}
            </NativeSelect.Field>
          </NativeSelect.Root>
          <Badge colorPalette={run.deathRate > 0.5 ? "red" : "green"}>
            死亡率 {(run.deathRate * 100).toFixed(1)}%
          </Badge>
          <Badge>到達深度中央値 {run.medianDepth}</Badge>
          <Badge>{run.trials}回試行</Badge>
        </Flex>

        <Stack gap="4">
          {chartSections.map((section) => (
            <Box key={section.title}>
              <Heading size="sm" mb="1">
                {section.title}
              </Heading>
              <ChartDescription
                model={section.model}
                criteria={section.criteria}
              />
              {section.render(run)}
            </Box>
          ))}

          <Table.Root size="sm">
            <Table.Header>
              <Table.Row>
                <Table.ColumnHeader>深度</Table.ColumnHeader>
                <Table.ColumnHeader>戦闘後HP</Table.ColumnHeader>
                <Table.ColumnHeader>回復後HP</Table.ColumnHeader>
                <Table.ColumnHeader>突然死率</Table.ColumnHeader>
                <Table.ColumnHeader>被ダメ</Table.ColumnHeader>
                <Table.ColumnHeader>回復量</Table.ColumnHeader>
                <Table.ColumnHeader>キルT</Table.ColumnHeader>
                <Table.ColumnHeader>武器Dmg</Table.ColumnHeader>
                <Table.ColumnHeader>空腹度</Table.ColumnHeader>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {run.depths.map((d) => (
                <Table.Row key={d.depth}>
                  <Table.Cell>{d.depth}</Table.Cell>
                  <Table.Cell>{d.medianHPBeforeHeal}</Table.Cell>
                  <Table.Cell>{d.medianHP}</Table.Cell>
                  <Table.Cell>
                    {(d.suddenDeathRate * 100).toFixed(1)}%
                  </Table.Cell>
                  <Table.Cell>{d.medianDamage}</Table.Cell>
                  <Table.Cell>{d.medianHealing}</Table.Cell>
                  <Table.Cell>{d.medianKillTurns}</Table.Cell>
                  <Table.Cell>{d.medianWeaponDamage}</Table.Cell>
                  <Table.Cell>{d.medianHunger}</Table.Cell>
                </Table.Row>
              ))}
            </Table.Body>
          </Table.Root>

          <TrialDetail run={run} playerHP={playerHP} />
        </Stack>
      </Box>
    </Stack>
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
