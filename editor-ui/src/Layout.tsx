import { Outlet, NavLink, useLocation } from "react-router-dom";
import { Box, Flex, Heading, Stack, Text } from "@chakra-ui/react";

interface NavSection {
  label: string;
  items: { path: string; label: string }[];
}

const sections: NavSection[] = [
  {
    label: "データ",
    items: [
      { path: "/items", label: "アイテム" },
      { path: "/members", label: "メンバー" },
      { path: "/props", label: "置物" },
      { path: "/recipes", label: "レシピ" },
      { path: "/professions", label: "職業" },
    ],
  },
  {
    label: "テーブル",
    items: [
      { path: "/command-tables", label: "コマンドテーブル" },
      { path: "/drop-tables", label: "ドロップテーブル" },
      { path: "/item-tables", label: "アイテムテーブル" },
      { path: "/enemy-tables", label: "敵テーブル" },
    ],
  },
  {
    label: "マップ",
    items: [
      { path: "/tiles", label: "タイル" },
      { path: "/palettes", label: "パレット" },
      { path: "/layouts", label: "レイアウト" },
    ],
  },
  {
    label: "グラフィック",
    items: [
      { path: "/sprite-sheets", label: "スプライトシート" },
      { path: "/cutter", label: "スプライト切り抜き" },
    ],
  },
  {
    label: "分析",
    items: [
      { path: "/balance", label: "バランス" },
      { path: "/dps", label: "DPS" },
      { path: "/table-viewer", label: "スポーンテーブル" },
    ],
  },
];

export function Layout() {
  const location = useLocation();

  return (
    <Flex h="100vh">
      <Box
        w="52"
        bg="bg.subtle"
        borderRight="1px solid"
        borderColor="border"
        p="3"
        overflowY="auto"
        flexShrink={0}
      >
        <Heading size="md" mb="4">
          Ruins Editor
        </Heading>
        <Stack gap="4">
          {sections.map((section, i) => (
            <Box key={section.label}>
              <Text
                fontSize="2xs"
                fontWeight="bold"
                color="fg.subtle"
                px="2"
                pb="1"
                mb="1"
                letterSpacing="wider"
                textTransform="uppercase"
                borderTop={i > 0 ? "1px solid" : undefined}
                borderColor="border"
                pt={i > 0 ? "3" : undefined}
              >
                {section.label}
              </Text>
              <Stack gap="0.5">
                {section.items.map((r) => (
                  <Box
                    key={r.path}
                    asChild
                    px="2"
                    py="1"
                    borderRadius="md"
                    fontSize="sm"
                    bg={
                      location.pathname.startsWith(r.path)
                        ? "bg.emphasized"
                        : undefined
                    }
                    _hover={{ bg: "bg.muted" }}
                  >
                    <NavLink to={r.path}>{r.label}</NavLink>
                  </Box>
                ))}
              </Stack>
            </Box>
          ))}
        </Stack>
      </Box>
      <Box flex="1" overflowY="auto" p="4">
        <Outlet />
      </Box>
    </Flex>
  );
}

export function WelcomePage() {
  return (
    <Text color="fg.muted">左のメニューからリソースを選択してください</Text>
  );
}
