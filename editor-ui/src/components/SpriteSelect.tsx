import { useEffect, useMemo, useRef, useState } from "react";
import { Box, Flex, Input, Text } from "@chakra-ui/react";
import { useSpriteSheet, type SpriteInfo } from "../hooks/useSprites";

interface SpriteSelectProps {
  sheetName: string;
  value: string;
  onChange: (key: string) => void;
}

const SPRITE_DISPLAY_SIZE = 32;

// スプライトプレビュー: CSS background-position でスプライトシートから切り出す
function SpritePreview({
  image,
  sprite,
  sheetWidth,
  sheetHeight,
  size = SPRITE_DISPLAY_SIZE,
}: {
  image: string;
  sprite: SpriteInfo;
  sheetWidth: number;
  sheetHeight: number;
  size?: number;
}) {
  const scale = size / sprite.w;
  return (
    <Box
      w={`${size}px`}
      h={`${size}px`}
      flexShrink={0}
      backgroundRepeat="no-repeat"
      imageRendering="pixelated"
      style={{
        backgroundImage: `url(${image})`,
        backgroundSize: `${sheetWidth * scale}px ${sheetHeight * scale}px`,
        backgroundPosition: `-${sprite.x * scale}px -${sprite.y * scale}px`,
      }}
    />
  );
}

export function SpriteSelect({ sheetName, value, onChange }: SpriteSelectProps) {
  const { data: sheet } = useSpriteSheet(sheetName);
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // 外部クリックで閉じる
  useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  const spriteMap = useMemo(() => {
    if (!sheet) return new Map<string, SpriteInfo>();
    return new Map(sheet.sprites.map((s) => [s.key, s]));
  }, [sheet]);

  const filtered = useMemo(() => {
    if (!sheet) return [];
    if (!search) return sheet.sprites;
    const lower = search.toLowerCase();
    return sheet.sprites.filter((s) => s.key.toLowerCase().includes(lower));
  }, [sheet, search]);

  const currentSprite = spriteMap.get(value);

  if (!sheet) {
    return (
      <Input
        size="sm"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="spriteKey"
      />
    );
  }

  return (
    <Box ref={containerRef} position="relative" flex="1">
      <Flex
        align="center"
        gap="2"
        borderWidth="1px"
        borderRadius="md"
        px="2"
        py="1"
        cursor="pointer"
        onClick={() => {
          setOpen(!open);
          setSearch("");
          setTimeout(() => inputRef.current?.focus(), 0);
        }}
        _hover={{ borderColor: "border.emphasized" }}
      >
        {currentSprite && sheet && (
          <SpritePreview image={sheet.image} sprite={currentSprite} sheetWidth={sheet.sheetWidth} sheetHeight={sheet.sheetHeight} size={24} />
        )}
        <Text fontSize="sm" flex="1" truncate>
          {value || "(未選択)"}
        </Text>
      </Flex>

      {open && (
        <Box
          position="absolute"
          top="100%"
          left="0"
          right="0"
          zIndex="10"
          bg="bg"
          borderWidth="1px"
          borderRadius="md"
          boxShadow="lg"
          mt="1"
          maxH="320px"
          overflow="hidden"
        >
          <Box p="1" borderBottom="1px solid" borderColor="border">
            <Input
              ref={inputRef}
              size="sm"
              placeholder="検索..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Escape") setOpen(false);
                if (e.key === "Enter" && filtered.length > 0) {
                  onChange(filtered[0]!.key);
                  setOpen(false);
                }
              }}
            />
          </Box>
          <Box overflowY="auto" maxH="272px">
            {filtered.length === 0 ? (
              <Text fontSize="sm" color="fg.muted" p="2">
                該当なし
              </Text>
            ) : (
              filtered.map((sprite) => (
                <Flex
                  key={sprite.key}
                  align="center"
                  gap="2"
                  px="2"
                  py="1"
                  cursor="pointer"
                  bg={sprite.key === value ? "bg.emphasized" : undefined}
                  _hover={{ bg: "bg.muted" }}
                  onClick={() => {
                    onChange(sprite.key);
                    setOpen(false);
                  }}
                >
                  <SpritePreview image={sheet.image} sprite={sprite} sheetWidth={sheet.sheetWidth} sheetHeight={sheet.sheetHeight} size={24} />
                  <Text fontSize="sm" truncate>
                    {sprite.key}
                  </Text>
                </Flex>
              ))
            )}
          </Box>
        </Box>
      )}
    </Box>
  );
}
