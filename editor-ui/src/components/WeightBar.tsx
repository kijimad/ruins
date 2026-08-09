import { useState } from "react";
import { Box, Flex, Text, Portal } from "@chakra-ui/react";

// WeightEntry は割合バーの1区画。weight の比で幅を決める。
export interface WeightEntry {
  name: string;
  weight: number;
}

// 色相を分散させて見分けやすい色を生成する
const barColors = [
  "hsl(210, 60%, 55%)",
  "hsl(340, 55%, 55%)",
  "hsl(150, 50%, 45%)",
  "hsl(30, 65%, 55%)",
  "hsl(270, 45%, 55%)",
  "hsl(180, 50%, 45%)",
  "hsl(60, 55%, 45%)",
  "hsl(0, 55%, 55%)",
  "hsl(120, 40%, 45%)",
  "hsl(300, 40%, 55%)",
  "hsl(45, 60%, 50%)",
  "hsl(195, 55%, 50%)",
];

// WeightBar は entries を weight 比で横に積み上げる割合バー。区画のホバーで名前と割合を出し、下に凡例を並べる。
// スポーンテーブルと部屋別 loot で共有する。
export function WeightBar({
  entries,
  totalWeight,
}: {
  entries: WeightEntry[];
  totalWeight: number;
}) {
  const [hover, setHover] = useState<{
    name: string;
    pct: string;
    x: number;
    y: number;
  } | null>(null);

  if (totalWeight === 0) return null;

  return (
    <Box>
      <Flex
        h="24px"
        borderRadius="md"
        overflow="hidden"
        w="100%"
        onMouseLeave={() => setHover(null)}
      >
        {entries.map((e, i) => {
          const pct = (e.weight / totalWeight) * 100;
          return (
            <Box
              key={e.name}
              style={{
                width: `${pct}%`,
                backgroundColor: barColors[i % barColors.length],
              }}
              minW="2px"
              cursor="default"
              onMouseEnter={(ev) =>
                setHover({
                  name: e.name,
                  pct: pct.toFixed(1),
                  x: ev.clientX,
                  y: ev.clientY,
                })
              }
              onMouseMove={(ev) =>
                setHover((prev) =>
                  prev ? { ...prev, x: ev.clientX, y: ev.clientY } : null,
                )
              }
            />
          );
        })}
      </Flex>
      {hover && (
        <Portal>
          <Box
            position="fixed"
            style={{ left: hover.x + 8, top: hover.y - 32 }}
            bg="bg.panel"
            borderWidth="1px"
            borderColor="border"
            borderRadius="md"
            px="2"
            py="1"
            boxShadow="md"
            zIndex="tooltip"
            pointerEvents="none"
          >
            <Text fontSize="xs" fontWeight="bold" whiteSpace="nowrap">
              {hover.name} ({hover.pct}%)
            </Text>
          </Box>
        </Portal>
      )}
      <Flex mt="1" gap="3" flexWrap="wrap">
        {entries.map((e, i) => (
          <Flex key={e.name} align="center" gap="1">
            <Box
              w="10px"
              h="10px"
              borderRadius="sm"
              flexShrink={0}
              style={{
                backgroundColor: barColors[i % barColors.length],
              }}
            />
            <Text fontSize="xs" color="fg.muted">
              {e.name} {((e.weight / totalWeight) * 100).toFixed(1)}%
            </Text>
          </Flex>
        ))}
      </Flex>
    </Box>
  );
}
