import { useEffect, useMemo, useRef } from "react";
import { Box } from "@chakra-ui/react";
import { useQuery } from "@tanstack/react-query";
import axios from "axios";
import { useResourceList } from "../hooks/useResource";
import { useSpriteSheet, type SpriteInfo, type SpriteSheetInfo } from "../hooks/useSprites";

// サーバー側で解決済みのセル
interface ResolvedCell {
  terrain: string;
  prop: string;
  npc: string;
}

// スプライト参照情報
interface SpriteRef {
  sheetName: string;
  spriteKey: string;
}

const TILE_SIZE = 16;

interface MapPreviewProps {
  layoutIndex: number;
  width: number;
  height: number;
  spawnPoints?: { x: number; y: number }[];
}

// サーバー側でplacements展開+パレット解決済みのセル配列を取得する
function useResolvedCells(layoutIndex: number) {
  return useQuery<ResolvedCell[][]>({
    queryKey: ["layouts", layoutIndex, "resolved"],
    queryFn: async () => {
      const res = await axios.get<{ cells: ResolvedCell[][] }>(`/api/v1/layouts/${layoutIndex}/resolved`);
      return res.data.cells;
    },
  });
}

export function MapPreview({ layoutIndex, width, height, spawnPoints }: MapPreviewProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  // サーバー側で解決済みのセル配列を取得する
  const resolvedQuery = useResolvedCells(layoutIndex);
  const resolvedCells = resolvedQuery.data ?? [];

  // タイル・置物・メンバーのデータを取得する（spriteKey/spriteSheetName の解決に使う）
  const tilesQuery = useResourceList<{ name: string; spriteRender: { spriteKey: string; spriteSheetName: string } }>("tiles");
  const propsQuery = useResourceList<{ name: string; spriteRender: { spriteKey: string; spriteSheetName: string } }>("props");
  const membersQuery = useResourceList<{ name: string; spriteKey: string; spriteSheetName: string }>("members");

  // タイル名→スプライト参照のマップを構築する
  const tileSpriteMap = useMemo(() => {
    const m = new Map<string, SpriteRef>();
    for (const tile of tilesQuery.data?.data ?? []) {
      m.set(tile.name, { sheetName: tile.spriteRender.spriteSheetName, spriteKey: tile.spriteRender.spriteKey });
    }
    return m;
  }, [tilesQuery.data]);

  // 置物名→スプライト参照のマップを構築する
  const propSpriteMap = useMemo(() => {
    const m = new Map<string, SpriteRef>();
    for (const prop of propsQuery.data?.data ?? []) {
      m.set(prop.name, { sheetName: prop.spriteRender.spriteSheetName, spriteKey: prop.spriteRender.spriteKey });
    }
    return m;
  }, [propsQuery.data]);

  // メンバー名→スプライト参照のマップを構築する
  const memberSpriteMap = useMemo(() => {
    const m = new Map<string, SpriteRef>();
    for (const member of membersQuery.data?.data ?? []) {
      m.set(member.name, { sheetName: member.spriteSheetName, spriteKey: member.spriteKey });
    }
    return m;
  }, [membersQuery.data]);

  // 必要なスプライトシート名を収集する
  const neededSheets = useMemo(() => {
    const sheets = new Set<string>();
    for (const row of resolvedCells) {
      for (const cell of row) {
        const tileRef = tileSpriteMap.get(cell.terrain);
        if (tileRef) sheets.add(tileRef.sheetName);
        const propRef = propSpriteMap.get(cell.prop);
        if (propRef) sheets.add(propRef.sheetName);
        const memberRef = memberSpriteMap.get(cell.npc);
        if (memberRef) sheets.add(memberRef.sheetName);
      }
    }
    return [...sheets];
  }, [resolvedCells, tileSpriteMap, propSpriteMap, memberSpriteMap]);

  // スプライトシートをフェッチする（最大5シート）
  const sheet0 = useSpriteSheet(neededSheets[0]);
  const sheet1 = useSpriteSheet(neededSheets[1]);
  const sheet2 = useSpriteSheet(neededSheets[2]);
  const sheet3 = useSpriteSheet(neededSheets[3]);
  const sheet4 = useSpriteSheet(neededSheets[4]);

  const sheetDataMap = useMemo(() => {
    const m = new Map<string, SpriteSheetInfo>();
    for (const q of [sheet0, sheet1, sheet2, sheet3, sheet4]) {
      if (q.data) m.set(q.data.name, q.data);
    }
    return m;
  }, [sheet0.data, sheet1.data, sheet2.data, sheet3.data, sheet4.data]);

  // スプライトキー→SpriteInfo の高速検索マップを構築する
  const spriteKeyMap = useMemo(() => {
    const m = new Map<string, { sheet: SpriteSheetInfo; sprite: SpriteInfo }>();
    for (const [, sheet] of sheetDataMap) {
      for (const sprite of sheet.sprites) {
        m.set(`${sheet.name}:${sprite.key}`, { sheet, sprite });
      }
    }
    return m;
  }, [sheetDataMap]);

  // 各シートの画像をロードする
  const loadedImages = useRef(new Map<string, HTMLImageElement>());

  useEffect(() => {
    let cancelled = false;
    const newImages: HTMLImageElement[] = [];
    for (const [name, info] of sheetDataMap) {
      if (!loadedImages.current.has(name)) {
        const img = new Image();
        img.src = info.image;
        img.onload = () => {
          if (cancelled) return;
          loadedImages.current.set(name, img);
          drawCanvas();
        };
        newImages.push(img);
      }
    }
    if (newImages.length === 0) drawCanvas();
    return () => { cancelled = true; };
  }, [sheetDataMap, resolvedCells, spawnPoints]);

  function findSprite(sheetName: string, spriteKey: string): { sheet: SpriteSheetInfo; sprite: SpriteInfo; img: HTMLImageElement } | undefined {
    // 完全一致を試し、なければ _0 サフィックス付きにフォールバックする
    // Go側では autoTileIndex で spriteKey に _N を付加するため
    const entry = spriteKeyMap.get(`${sheetName}:${spriteKey}`)
      ?? spriteKeyMap.get(`${sheetName}:${spriteKey}_0`);
    if (!entry) return undefined;
    const img = loadedImages.current.get(sheetName);
    if (!img) return undefined;
    return { sheet: entry.sheet, sprite: entry.sprite, img };
  }

  function drawCanvas() {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    ctx.imageSmoothingEnabled = false;
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    // 背景を黒で塗りつぶす
    ctx.fillStyle = "#111";
    ctx.fillRect(0, 0, canvas.width, canvas.height);

    for (let y = 0; y < resolvedCells.length; y++) {
      const row = resolvedCells[y]!;
      for (let x = 0; x < row.length; x++) {
        const cell = row[x]!;
        const dx = x * TILE_SIZE;
        const dy = y * TILE_SIZE;

        // 地形を描画する
        if (cell.terrain) {
          const tileRef = tileSpriteMap.get(cell.terrain);
          if (tileRef) {
            const found = findSprite(tileRef.sheetName, tileRef.spriteKey);
            if (found) {
              ctx.drawImage(found.img, found.sprite.x, found.sprite.y, found.sprite.w, found.sprite.h, dx, dy, TILE_SIZE, TILE_SIZE);
            }
          }
        }

        // 置物を描画する（地形の上に重ねる）
        if (cell.prop) {
          const propRef = propSpriteMap.get(cell.prop);
          if (propRef) {
            const found = findSprite(propRef.sheetName, propRef.spriteKey);
            if (found) {
              ctx.drawImage(found.img, found.sprite.x, found.sprite.y, found.sprite.w, found.sprite.h, dx, dy, TILE_SIZE, TILE_SIZE);
            }
          }
        }

        // NPCを描画する（地形の上に重ねる）
        if (cell.npc) {
          const memberRef = memberSpriteMap.get(cell.npc);
          if (memberRef) {
            const found = findSprite(memberRef.sheetName, memberRef.spriteKey);
            if (found) {
              ctx.drawImage(found.img, found.sprite.x, found.sprite.y, found.sprite.w, found.sprite.h, dx, dy, TILE_SIZE, TILE_SIZE);
            }
          }
        }
      }
    }

    // スポーン地点を描画する
    if (spawnPoints) {
      ctx.strokeStyle = "#0f0";
      ctx.lineWidth = 2;
      for (const sp of spawnPoints) {
        ctx.strokeRect(sp.x * TILE_SIZE + 1, sp.y * TILE_SIZE + 1, TILE_SIZE - 2, TILE_SIZE - 2);
      }
    }
  }

  // 解決済みセルからサイズを決定する（placementsで子チャンクが展開された場合、元のwidthより大きくなりうる）
  const actualW = resolvedCells[0]?.length ?? width;
  const actualH = resolvedCells.length || height;
  const canvasW = actualW * TILE_SIZE;
  const canvasH = actualH * TILE_SIZE;

  return (
    <Box maxW="100%" borderWidth="1px" borderRadius="md">
      <canvas
        ref={canvasRef}
        width={canvasW}
        height={canvasH}
        style={{ imageRendering: "pixelated" }}
      />
    </Box>
  );
}
