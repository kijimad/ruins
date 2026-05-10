import { render } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ChakraProvider, defaultSystem } from "@chakra-ui/react";
import { MapPreview } from "./MapPreview";
import { vi } from "vitest";

// Canvas APIのモック
const mockDrawImage = vi.fn();
const mockFillRect = vi.fn();
const mockClearRect = vi.fn();
const mockStrokeRect = vi.fn();

HTMLCanvasElement.prototype.getContext = vi.fn().mockReturnValue({
  drawImage: mockDrawImage,
  fillRect: mockFillRect,
  clearRect: mockClearRect,
  strokeRect: mockStrokeRect,
  imageSmoothingEnabled: true,
  fillStyle: "",
  strokeStyle: "",
  lineWidth: 1,
}) as unknown as typeof HTMLCanvasElement.prototype.getContext;

vi.mock("../hooks/useResource", () => ({
  useResourceList: vi.fn(() => ({
    data: { data: [] },
    isLoading: false,
    error: null,
  })),
}));

vi.mock("../hooks/useSprites", () => ({
  useSpriteSheet: vi.fn(() => ({ data: undefined })),
}));

vi.mock("@tanstack/react-query", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@tanstack/react-query")>();
  return {
    ...actual,
    useQuery: vi.fn(() => ({
      data: undefined,
      isLoading: false,
      error: null,
    })),
  };
});

import { useQuery } from "@tanstack/react-query";
import { useResourceList } from "../hooks/useResource";
import { useSpriteSheet } from "../hooks/useSprites";

const mockedUseQuery = vi.mocked(useQuery);
const mockedUseResourceList = vi.mocked(useResourceList);
const mockedUseSpriteSheet = vi.mocked(useSpriteSheet);

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return (
    <ChakraProvider value={defaultSystem}>
      <QueryClientProvider client={qc}>{children}</QueryClientProvider>
    </ChakraProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();

  HTMLCanvasElement.prototype.getContext = vi.fn().mockReturnValue({
    drawImage: mockDrawImage,
    fillRect: mockFillRect,
    clearRect: mockClearRect,
    strokeRect: mockStrokeRect,
    imageSmoothingEnabled: true,
    fillStyle: "",
    strokeStyle: "",
    lineWidth: 1,
  }) as unknown as typeof HTMLCanvasElement.prototype.getContext;

  mockedUseQuery.mockReturnValue({
    data: undefined,
    isLoading: false,
    error: null,
  } as any);
  mockedUseResourceList.mockReturnValue({
    data: { data: [] },
    isLoading: false,
    error: null,
  } as any);
  mockedUseSpriteSheet.mockReturnValue({ data: undefined } as any);
});

describe("MapPreview", () => {
  test("canvasが正しいサイズで描画される", () => {
    const { container } = render(
      <MapPreview layoutIndex={0} width={10} height={8} />,
      { wrapper },
    );
    const canvas = container.querySelector("canvas");
    expect(canvas).not.toBeNull();
    // resolvedCellsが空のときはwidth/heightのpropsから算出される
    expect(canvas!.width).toBe(10 * 16);
    expect(canvas!.height).toBe(8 * 16);
  });

  test("resolvedCellsがあるときはセル配列のサイズに従う", () => {
    const cells = [
      [
        { terrain: "floor", prop: "", npc: "" },
        { terrain: "wall", prop: "", npc: "" },
        { terrain: "floor", prop: "", npc: "" },
      ],
      [
        { terrain: "wall", prop: "", npc: "" },
        { terrain: "floor", prop: "chest", npc: "" },
        { terrain: "wall", prop: "", npc: "" },
      ],
    ];
    mockedUseQuery.mockReturnValue({
      data: cells,
      isLoading: false,
      error: null,
    } as any);

    const { container } = render(
      <MapPreview layoutIndex={0} width={1} height={1} />,
      { wrapper },
    );
    const canvas = container.querySelector("canvas");
    // resolvedCellsのサイズ(3x2)に基づく
    expect(canvas!.width).toBe(3 * 16);
    expect(canvas!.height).toBe(2 * 16);
  });

  test("スポーン地点の矩形が描画される", () => {
    const cells = [
      [{ terrain: "floor", prop: "", npc: "" }],
      [{ terrain: "floor", prop: "", npc: "" }],
    ];
    mockedUseQuery.mockReturnValue({
      data: cells,
      isLoading: false,
      error: null,
    } as any);

    render(
      <MapPreview
        layoutIndex={0}
        width={1}
        height={2}
        spawnPoints={[
          { x: 0, y: 0 },
          { x: 0, y: 1 },
        ]}
      />,
      { wrapper },
    );
    expect(mockStrokeRect).toHaveBeenCalledTimes(2);
  });

  test("タイルデータがあるとき必要なスプライトシートが取得される", () => {
    const cells = [[{ terrain: "stone_floor", prop: "", npc: "" }]];
    mockedUseQuery.mockReturnValue({
      data: cells,
      isLoading: false,
      error: null,
    } as any);

    mockedUseResourceList.mockImplementation((resource: string) => {
      if (resource === "tiles") {
        return {
          data: {
            data: [
              {
                name: "stone_floor",
                spriteRender: {
                  spriteKey: "floor_0",
                  spriteSheetName: "dungeon",
                },
              },
            ],
          },
          isLoading: false,
          error: null,
        } as any;
      }
      return {
        data: { data: [] },
        isLoading: false,
        error: null,
      } as any;
    });

    render(<MapPreview layoutIndex={0} width={1} height={1} />, { wrapper });

    // タイルの spriteSheetName="dungeon" に対して useSpriteSheet が呼ばれる
    expect(mockedUseSpriteSheet).toHaveBeenCalledWith("dungeon");
  });

  test("spawnPointsが未指定のときstrokeRectは呼ばれない", () => {
    const cells = [[{ terrain: "floor", prop: "", npc: "" }]];
    mockedUseQuery.mockReturnValue({
      data: cells,
      isLoading: false,
      error: null,
    } as any);

    render(<MapPreview layoutIndex={0} width={1} height={1} />, { wrapper });
    expect(mockStrokeRect).not.toHaveBeenCalled();
  });
});
