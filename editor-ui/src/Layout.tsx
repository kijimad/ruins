import { Outlet, NavLink, useLocation } from "react-router-dom";
import { Box, Flex, Heading, Stack, Text } from "@chakra-ui/react";

const resources = [
  { path: "/items", label: "アイテム" },
  { path: "/members", label: "メンバー" },
  { path: "/recipes", label: "レシピ" },
  { path: "/command-tables", label: "コマンドテーブル" },
  { path: "/drop-tables", label: "ドロップテーブル" },
  { path: "/item-tables", label: "アイテムテーブル" },
  { path: "/enemy-tables", label: "敵テーブル" },
  { path: "/tiles", label: "タイル" },
  { path: "/props", label: "置物" },
  { path: "/professions", label: "職業" },
  { path: "/sprite-sheets", label: "スプライトシート" },
  { path: "/palettes", label: "パレット" },
];

export function Layout() {
  const location = useLocation();

  return (
    <Flex h="100vh">
      <Box
        w="200px"
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
        <Stack gap="1">
          {resources.map((r) => (
            <Box
              key={r.path}
              asChild
              px="2"
              py="1"
              borderRadius="md"
              fontSize="sm"
              bg={location.pathname.startsWith(r.path) ? "bg.emphasized" : undefined}
              _hover={{ bg: "bg.muted" }}
            >
              <NavLink to={r.path}>{r.label}</NavLink>
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
  return <Text color="fg.muted">左のメニューからリソースを選択してください</Text>;
}
