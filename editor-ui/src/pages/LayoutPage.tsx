import { useEffect, useMemo, useRef, useState } from "react";
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
  Textarea,
} from "@chakra-ui/react";
import {
  useResourceList,
  useResourceUpdate,
  useResourceCreate,
  useResourceDelete,
} from "../hooks/useResource";
import { SearchableSelect } from "../components/SearchableSelect";
import { MapPreview } from "../components/MapPreview";

interface SpawnPoint {
  x: number;
  y: number;
}

interface Placement {
  id: string;
  chunks: string[];
}

interface LayoutChunk {
  name: string;
  weight: number;
  palettes: string[];
  map: string;
  Size: { W: number; H: number };
  spawn_points: SpawnPoint[];
  placements: Placement[];
}

export function LayoutPage() {
  const { data, isLoading, error } = useResourceList<LayoutChunk>("layouts");
  const updateResource = useResourceUpdate<LayoutChunk>("layouts");
  const createResource = useResourceCreate<LayoutChunk>("layouts");
  const deleteResource = useResourceDelete("layouts");
  const palettesQuery = useResourceList<{ id: string }>("palettes");

  const [selectedIndex, setSelectedIndex] = useState<number | null>(null);
  const [editData, setEditData] = useState<LayoutChunk | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [confirmDeleteIndex, setConfirmDeleteIndex] = useState<number | null>(
    null,
  );
  const listRef = useRef<HTMLDivElement>(null);

  const items = useMemo(() => data?.data ?? [], [data]);
  const paletteNames = palettesQuery.data?.data?.map((p) => p.id) ?? [];
  // レイアウト名一覧（placements の chunks 参照用）
  const layoutNames = items.map((item) => item.name);

  useEffect(() => {
    if (selectedIndex !== null && items[selectedIndex]) {
      setEditData(structuredClone(items[selectedIndex]) as LayoutChunk);
    }
  }, [items, selectedIndex]);

  if (isLoading) return <Text>読み込み中...</Text>;
  if (error) return <Text color="red.500">エラー: {String(error)}</Text>;

  function handleCreate() {
    const template: LayoutChunk = {
      name: "new_layout",
      weight: 100,
      palettes: ["standard"],
      map: "##########\n#........#\n#........#\n#........#\n#........#\n##########\n",
      Size: { W: 10, H: 6 },
      spawn_points: [{ x: 1, y: 1 }],
      placements: [],
    };
    createResource.mutate(template, {
      onSuccess: (result) => {
        setSelectedIndex(result.index);
        setSaveError(null);
      },
    });
  }

  function handleSelect(index: number) {
    setSelectedIndex(index);
    setEditData(structuredClone(items[index]) as LayoutChunk);
    setSaveError(null);
    setSaveSuccess(false);
  }

  function handleSave() {
    if (selectedIndex === null || !editData) return;
    setSaveError(null);
    setSaveSuccess(false);
    updateResource.mutate(
      { index: selectedIndex, data: editData },
      {
        onSuccess: () => {
          setSaveSuccess(true);
          setTimeout(() => setSaveSuccess(false), 2000);
        },
        onError: (err) => setSaveError(String(err)),
      },
    );
  }

  function handleDelete(index: number) {
    if (confirmDeleteIndex === index) {
      setConfirmDeleteIndex(null);
      deleteResource.mutate(index, {
        onSuccess: () => {
          if (selectedIndex === index) {
            setSelectedIndex(null);
            setEditData(null);
          }
        },
      });
    } else {
      setConfirmDeleteIndex(index);
      setTimeout(
        () => setConfirmDeleteIndex((prev) => (prev === index ? null : prev)),
        3000,
      );
    }
  }

  function update(fn: (d: LayoutChunk) => void) {
    if (!editData) return;
    const next = structuredClone(editData);
    fn(next);
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
            レイアウト
            <Badge ml="2" colorPalette="gray">
              {items.length}
            </Badge>
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
                {item.name}
              </Text>
              <Button
                size="xs"
                variant={confirmDeleteIndex === index ? "solid" : "ghost"}
                colorPalette="red"
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(index);
                }}
              >
                {confirmDeleteIndex === index ? "本当に?" : "×"}
              </Button>
            </Flex>
          ))}
        </Stack>
      </Box>

      {/* 編集エリア */}
      <Box flex="1" overflowY="auto">
        {editData ? (
          <>
            <Flex
              justify="space-between"
              align="center"
              mb="4"
              position="sticky"
              top="0"
              bg="bg"
              zIndex="1"
              py="2"
            >
              <Heading size="md">{editData.name}</Heading>
              <Flex align="center" gap="2">
                {saveSuccess && (
                  <Text fontSize="sm" color="green.500" fontWeight="bold">
                    保存しました
                  </Text>
                )}
                <Button
                  size="sm"
                  colorPalette="blue"
                  onClick={handleSave}
                  loading={updateResource.isPending}
                >
                  保存
                </Button>
              </Flex>
            </Flex>
            {saveError && (
              <Text color="red.500" fontSize="sm" mb="2">
                {saveError}
              </Text>
            )}

            <Stack gap="3">
              {/* 基本情報 */}
              <Flex align="center" gap="3">
                <Text fontSize="sm" w="120px" flexShrink={0} color="fg.muted">
                  name
                </Text>
                <Input
                  size="sm"
                  value={editData.name}
                  onChange={(e) =>
                    update((d) => {
                      d.name = e.target.value;
                    })
                  }
                />
              </Flex>
              <Flex align="center" gap="3">
                <Text fontSize="sm" w="120px" flexShrink={0} color="fg.muted">
                  weight
                </Text>
                <Input
                  size="sm"
                  type="number"
                  value={editData.weight}
                  onChange={(e) =>
                    update((d) => {
                      d.weight = parseInt(e.target.value, 10) || 0;
                    })
                  }
                />
              </Flex>
              <Flex align="center" gap="3">
                <Text fontSize="sm" w="120px" flexShrink={0} color="fg.muted">
                  Size W
                </Text>
                <Input
                  size="sm"
                  type="number"
                  value={editData.Size.W}
                  onChange={(e) =>
                    update((d) => {
                      d.Size.W = parseInt(e.target.value, 10) || 0;
                    })
                  }
                />
              </Flex>
              <Flex align="center" gap="3">
                <Text fontSize="sm" w="120px" flexShrink={0} color="fg.muted">
                  Size H
                </Text>
                <Input
                  size="sm"
                  type="number"
                  value={editData.Size.H}
                  onChange={(e) =>
                    update((d) => {
                      d.Size.H = parseInt(e.target.value, 10) || 0;
                    })
                  }
                />
              </Flex>

              {/* パレット */}
              <Fieldset.Root borderWidth="1px" borderRadius="md" p="3">
                <Fieldset.Legend fontSize="sm" fontWeight="bold" px="1">
                  palettes ({editData.palettes.length})
                </Fieldset.Legend>
                <Stack gap="1">
                  {editData.palettes.map((pal, i) => (
                    <Flex key={i} gap="1" align="center">
                      <SearchableSelect
                        options={paletteNames}
                        value={pal}
                        onChange={(v) =>
                          update((d) => {
                            d.palettes[i] = v;
                          })
                        }
                      />
                      <Button
                        size="xs"
                        variant="ghost"
                        colorPalette="red"
                        onClick={() =>
                          update((d) => {
                            d.palettes.splice(i, 1);
                          })
                        }
                      >
                        ×
                      </Button>
                    </Flex>
                  ))}
                  <Button
                    size="xs"
                    variant="outline"
                    onClick={() =>
                      update((d) => {
                        d.palettes.push("");
                      })
                    }
                  >
                    ＋ 追加
                  </Button>
                </Stack>
              </Fieldset.Root>

              {/* スポーン地点 */}
              <Fieldset.Root borderWidth="1px" borderRadius="md" p="3">
                <Fieldset.Legend fontSize="sm" fontWeight="bold" px="1">
                  spawn_points ({editData.spawn_points.length})
                </Fieldset.Legend>
                <Stack gap="2">
                  {editData.spawn_points.map((sp, i) => (
                    <Flex key={i} gap="2" align="center">
                      <Text fontSize="xs" color="fg.subtle" w="24px">
                        #{i}
                      </Text>
                      <Text fontSize="sm" color="fg.muted" flexShrink={0}>
                        x
                      </Text>
                      <Input
                        size="sm"
                        type="number"
                        w="80px"
                        value={sp.x}
                        onChange={(e) =>
                          update((d) => {
                            d.spawn_points[i]!.x =
                              parseInt(e.target.value, 10) || 0;
                          })
                        }
                      />
                      <Text fontSize="sm" color="fg.muted" flexShrink={0}>
                        y
                      </Text>
                      <Input
                        size="sm"
                        type="number"
                        w="80px"
                        value={sp.y}
                        onChange={(e) =>
                          update((d) => {
                            d.spawn_points[i]!.y =
                              parseInt(e.target.value, 10) || 0;
                          })
                        }
                      />
                      <Button
                        size="xs"
                        variant="ghost"
                        colorPalette="red"
                        onClick={() =>
                          update((d) => {
                            d.spawn_points.splice(i, 1);
                          })
                        }
                      >
                        ×
                      </Button>
                    </Flex>
                  ))}
                  <Button
                    size="xs"
                    variant="outline"
                    onClick={() =>
                      update((d) => {
                        d.spawn_points.push({ x: 0, y: 0 });
                      })
                    }
                  >
                    ＋ 追加
                  </Button>
                </Stack>
              </Fieldset.Root>

              {/* プレースメント */}
              <Fieldset.Root borderWidth="1px" borderRadius="md" p="3">
                <Fieldset.Legend fontSize="sm" fontWeight="bold" px="1">
                  placements ({editData.placements.length})
                </Fieldset.Legend>
                <Stack gap="3">
                  {editData.placements.map((pl, i) => (
                    <Box key={i} borderWidth="1px" borderRadius="md" p="2">
                      <Flex justify="space-between" align="center" mb="1">
                        <Text fontSize="xs" color="fg.subtle">
                          #{i}
                        </Text>
                        <Button
                          size="xs"
                          variant="ghost"
                          colorPalette="red"
                          onClick={() =>
                            update((d) => {
                              d.placements.splice(i, 1);
                            })
                          }
                        >
                          ×
                        </Button>
                      </Flex>
                      <Stack gap="1">
                        <Flex align="center" gap="3">
                          <Text
                            fontSize="sm"
                            w="80px"
                            flexShrink={0}
                            color="fg.muted"
                          >
                            id
                          </Text>
                          <Input
                            size="sm"
                            value={pl.id}
                            onChange={(e) =>
                              update((d) => {
                                d.placements[i]!.id = e.target.value;
                              })
                            }
                          />
                        </Flex>
                        <Flex align="start" gap="3">
                          <Text
                            fontSize="sm"
                            w="80px"
                            flexShrink={0}
                            color="fg.muted"
                            pt="1"
                          >
                            chunks
                          </Text>
                          <Stack gap="1" flex="1">
                            {pl.chunks.map((ch, j) => (
                              <Flex key={j} gap="1">
                                <SearchableSelect
                                  options={layoutNames}
                                  value={ch}
                                  onChange={(v) =>
                                    update((d) => {
                                      d.placements[i]!.chunks[j] = v;
                                    })
                                  }
                                />
                                <Button
                                  size="xs"
                                  variant="ghost"
                                  colorPalette="red"
                                  onClick={() =>
                                    update((d) => {
                                      d.placements[i]!.chunks.splice(j, 1);
                                    })
                                  }
                                >
                                  ×
                                </Button>
                              </Flex>
                            ))}
                            <Button
                              size="xs"
                              variant="outline"
                              onClick={() =>
                                update((d) => {
                                  d.placements[i]!.chunks.push("");
                                })
                              }
                            >
                              ＋ 追加
                            </Button>
                          </Stack>
                        </Flex>
                      </Stack>
                    </Box>
                  ))}
                  <Button
                    size="xs"
                    variant="outline"
                    onClick={() =>
                      update((d) => {
                        d.placements.push({ id: "", chunks: [] });
                      })
                    }
                  >
                    ＋ 追加
                  </Button>
                </Stack>
              </Fieldset.Root>

              {/* マップ */}
              <Fieldset.Root borderWidth="1px" borderRadius="md" p="3">
                <Fieldset.Legend fontSize="sm" fontWeight="bold" px="1">
                  map ({editData.Size.W}x{editData.Size.H})
                </Fieldset.Legend>
                <Textarea
                  fontFamily="monospace"
                  fontSize="xs"
                  lineHeight="1.2"
                  rows={Math.min(editData.Size.H + 2, 60)}
                  value={editData.map.replace(/^\n/, "").replace(/\n$/, "")}
                  onChange={(e) =>
                    update((d) => {
                      d.map = e.target.value + "\n";
                    })
                  }
                  spellCheck={false}
                  resize="vertical"
                />
              </Fieldset.Root>

              {/* マッププレビュー */}
              {selectedIndex !== null && (
                <Fieldset.Root borderWidth="1px" borderRadius="md" p="3">
                  <Fieldset.Legend fontSize="sm" fontWeight="bold" px="1">
                    preview
                  </Fieldset.Legend>
                  <MapPreview
                    layoutIndex={selectedIndex}
                    width={editData.Size.W}
                    height={editData.Size.H}
                    spawnPoints={editData.spawn_points}
                  />
                </Fieldset.Root>
              )}
            </Stack>
          </>
        ) : (
          <Text color="fg.muted">左の一覧からレイアウトを選択してください</Text>
        )}
      </Box>
    </Flex>
  );
}
