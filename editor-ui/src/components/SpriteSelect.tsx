import { useMemo } from "react";
import { Box, Input } from "@chakra-ui/react";
import { useSpriteSheet, type SpriteInfo } from "../hooks/useSprites";
import { SearchableSelect } from "./SearchableSelect";

interface SpriteSelectProps {
  sheetName: string;
  value: string;
  onChange: (key: string) => void;
}

const SPRITE_DISPLAY_SIZE = 24;

// スプライトプレビュー: CSS background-position でスプライトシートから切り出す
export function SpritePreview({
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

  const spriteMap = useMemo(() => {
    if (!sheet) return new Map<string, SpriteInfo>();
    return new Map(sheet.sprites.map((s) => [s.key, s]));
  }, [sheet]);

  const spriteKeys = useMemo(
    () => (sheet ? sheet.sprites.map((s) => s.key) : []),
    [sheet],
  );

  if (!sheet) {
    return (
      <Input size="sm" value={value} disabled placeholder="読み込み中..." />
    );
  }

  const renderSprite = (key: string) => {
    const sprite = spriteMap.get(key);
    if (!sprite) return null;
    return (
      <SpritePreview
        image={sheet.image}
        sprite={sprite}
        sheetWidth={sheet.sheetWidth}
        sheetHeight={sheet.sheetHeight}
      />
    );
  };

  return (
    <SearchableSelect
      options={spriteKeys}
      value={value}
      onChange={onChange}
      placeholder="(未選択)"
      renderSelected={renderSprite}
      renderItem={renderSprite}
    />
  );
}
