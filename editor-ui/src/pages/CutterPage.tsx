import { useCallback, useRef, useState } from "react";
import { Box, Flex, Heading, Input, Text } from "@chakra-ui/react";

const CELL_SIZE = 32;
const DISPLAY_SCALE = 2;

interface CellData {
  index: number;
  row: number;
  col: number;
  name: string;
  transparent: boolean;
}

export function CutterPage() {
  const [image, setImage] = useState<HTMLImageElement | null>(null);
  const [cells, setCells] = useState<CellData[]>([]);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState("");
  const canvasRefs = useRef<Map<number, HTMLCanvasElement>>(new Map());

  const handleUpload = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      const img = new Image();
      img.onload = () => {
        setImage(img);
        const c = Math.floor(img.width / CELL_SIZE);
        const r = Math.floor(img.height / CELL_SIZE);

        // 各セルが透明かどうか判定する
        const tempCanvas = document.createElement("canvas");
        tempCanvas.width = img.width;
        tempCanvas.height = img.height;
        const ctx = tempCanvas.getContext("2d")!;
        ctx.drawImage(img, 0, 0);

        const newCells: CellData[] = [];
        for (let row = 0; row < r; row++) {
          for (let col = 0; col < c; col++) {
            const idx = row * c + col;
            const imageData = ctx.getImageData(
              col * CELL_SIZE,
              row * CELL_SIZE,
              CELL_SIZE,
              CELL_SIZE,
            );
            const transparent = isTransparent(imageData);
            newCells.push({
              index: idx,
              row,
              col,
              name: "",
              transparent,
            });
          }
        }
        setCells(newCells);
        setMessage("");
      };
      img.src = reader.result as string;
    };
    reader.readAsDataURL(file);
  }, []);

  // セルのcanvasにスプライトを描画する
  const drawCell = useCallback(
    (canvas: HTMLCanvasElement | null, cell: CellData) => {
      if (!canvas || !image) return;
      canvasRefs.current.set(cell.index, canvas);
      canvas.width = CELL_SIZE * DISPLAY_SCALE;
      canvas.height = CELL_SIZE * DISPLAY_SCALE;
      const ctx = canvas.getContext("2d");
      if (!ctx) return;
      ctx.imageSmoothingEnabled = false;
      ctx.clearRect(0, 0, canvas.width, canvas.height);
      ctx.drawImage(
        image,
        cell.col * CELL_SIZE,
        cell.row * CELL_SIZE,
        CELL_SIZE,
        CELL_SIZE,
        0,
        0,
        CELL_SIZE * DISPLAY_SCALE,
        CELL_SIZE * DISPLAY_SCALE,
      );
    },
    [image],
  );

  const handleNameChange = (index: number, name: string) => {
    setCells((prev) =>
      prev.map((c) => (c.index === index ? { ...c, name } : c)),
    );
  };

  // 名前が入力されたセルをサーバーのsingleディレクトリに保存する
  const handleSave = async () => {
    if (!image) return;
    const named = cells.filter((c) => c.name.trim() !== "");
    if (named.length === 0) {
      setMessage("保存するスプライトがありません。名前を入力してください");
      return;
    }

    setSaving(true);
    setMessage("");

    const sprites: { name: string; data: string }[] = [];
    for (const cell of named) {
      const canvas = document.createElement("canvas");
      canvas.width = CELL_SIZE;
      canvas.height = CELL_SIZE;
      const ctx = canvas.getContext("2d");
      if (!ctx) continue;
      ctx.imageSmoothingEnabled = false;
      ctx.drawImage(
        image,
        cell.col * CELL_SIZE,
        cell.row * CELL_SIZE,
        CELL_SIZE,
        CELL_SIZE,
        0,
        0,
        CELL_SIZE,
        CELL_SIZE,
      );
      sprites.push({
        name: cell.name.trim(),
        data: canvas.toDataURL("image/png"),
      });
    }

    try {
      const res = await fetch("/api/v1/cutter/save", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ sprites }),
      });
      const result = await res.json();
      setMessage(result.message);
    } catch (e) {
      setMessage(`保存に失敗: ${e instanceof Error ? e.message : String(e)}`);
    }
    setSaving(false);
  };

  return (
    <Box>
      <Heading size="lg" mb="4">
        スプライト切り抜き
      </Heading>

      <Box mb="4">
        <Text fontSize="sm" mb="1">
          スプライトシート画像をアップロードすると、32x32のグリッドに分割されます。
          各セルに名前を入力して保存すると、個別のPNGファイルとして書き出します。
        </Text>
        <Input
          type="file"
          accept="image/png"
          size="sm"
          onChange={handleUpload}
          maxW="80"
        />
      </Box>

      {image && (
        <>
          <Flex gap="3" mb="4" align="center">
            <Text fontSize="sm" color="fg.muted">
              {Math.floor(image.width / CELL_SIZE)}x
              {Math.floor(image.height / CELL_SIZE)} セル (
              {cells.filter((c) => !c.transparent).length} 個が非透明)
            </Text>
            <Box
              as="button"
              px="3"
              py="1"
              fontSize="sm"
              borderRadius="md"
              borderWidth="1px"
              _hover={{ bg: "bg.muted" }}
              onClick={handleSave}
              opacity={saving ? 0.5 : 1}
              pointerEvents={saving ? "none" : "auto"}
            >
              {saving ? "保存中..." : "PNG保存"}
            </Box>
            {message && (
              <Text fontSize="sm" color="green.500">
                {message}
              </Text>
            )}
          </Flex>

          <Box overflowY="auto" maxH="calc(100vh - 250px)">
            <Flex flexWrap="wrap" gap="2">
              {cells
                .filter((c) => !c.transparent)
                .map((cell) => (
                  <Flex
                    key={cell.index}
                    direction="column"
                    align="center"
                    gap="1"
                  >
                    <canvas
                      ref={(el) => drawCell(el, cell)}
                      style={{
                        imageRendering: "pixelated",
                        border: "1px solid var(--chakra-colors-border)",
                        borderRadius: "2px",
                      }}
                    />
                    <Input
                      size="xs"
                      w={`${CELL_SIZE * DISPLAY_SCALE + 16}px`}
                      placeholder={`${cell.row},${cell.col}`}
                      value={cell.name}
                      onChange={(e) =>
                        handleNameChange(cell.index, e.target.value)
                      }
                      fontSize="xs"
                    />
                  </Flex>
                ))}
            </Flex>
          </Box>
        </>
      )}
    </Box>
  );
}

function isTransparent(imageData: ImageData): boolean {
  const data = imageData.data;
  for (let i = 3; i < data.length; i += 4) {
    if (data[i]! > 0) return false;
  }
  return true;
}
