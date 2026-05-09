import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ChakraProvider, defaultSystem } from "@chakra-ui/react";
import { LayoutPage } from "./LayoutPage";
import { vi } from "vitest";

const mockMutate = vi.fn();
const mockCreateMutate = vi.fn();
const mockDeleteMutate = vi.fn();

vi.mock("../hooks/useResource", () => ({
  useResourceList: vi.fn(),
  useResourceUpdate: vi.fn(() => ({ mutate: mockMutate, isPending: false })),
  useResourceCreate: vi.fn(() => ({
    mutate: mockCreateMutate,
    isPending: false,
  })),
  useResourceDelete: vi.fn(() => ({
    mutate: mockDeleteMutate,
    isPending: false,
  })),
}));

vi.mock("../components/MapPreview", () => ({
  MapPreview: ({ layoutIndex }: { layoutIndex: number }) => (
    <div data-testid="map-preview">preview-{layoutIndex}</div>
  ),
}));

import { useResourceList } from "../hooks/useResource";
const mockedUseResourceList = vi.mocked(useResourceList);

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <ChakraProvider value={defaultSystem}>
      <QueryClientProvider client={qc}>{children}</QueryClientProvider>
    </ChakraProvider>
  );
}

const layoutsData = {
  data: [
    {
      name: "dungeon_a",
      weight: 100,
      palettes: ["standard"],
      map: "####\n#..#\n####\n",
      Size: { W: 4, H: 3 },
      spawn_points: [{ x: 1, y: 1 }],
      placements: [],
    },
    {
      name: "dungeon_b",
      weight: 50,
      palettes: ["cave"],
      map: "######\n#....#\n######\n",
      Size: { W: 6, H: 3 },
      spawn_points: [{ x: 2, y: 1 }],
      placements: [{ id: "room1", chunks: ["sub_chunk"] }],
    },
  ],
  totalCount: 2,
};

const palettesData = {
  data: [{ id: "standard" }, { id: "cave" }, { id: "forest" }],
  totalCount: 3,
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const layoutsResult: any = { data: layoutsData, isLoading: false, error: null };
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const palettesResult: any = {
  data: palettesData,
  isLoading: false,
  error: null,
};
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const emptyResult: any = { data: null, isLoading: false, error: null };

function setupLists() {
  mockedUseResourceList.mockImplementation((resource: string) => {
    if (resource === "layouts") return layoutsResult;
    if (resource === "palettes") return palettesResult;
    return emptyResult;
  });
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("LayoutPage", () => {
  test("一覧にレイアウト名が表示される", () => {
    setupLists();
    render(<LayoutPage />, { wrapper });

    expect(screen.getByText("dungeon_a")).toBeInTheDocument();
    expect(screen.getByText("dungeon_b")).toBeInTheDocument();
  });

  test("ローディング中は読み込み中テキストが表示される", () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockedUseResourceList.mockReturnValue({
      data: null,
      isLoading: true,
      error: null,
    } as any);
    render(<LayoutPage />, { wrapper });
    expect(screen.getByText("読み込み中...")).toBeInTheDocument();
  });

  test("エラー時はエラーメッセージが表示される", () => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    mockedUseResourceList.mockReturnValue({
      data: null,
      isLoading: false,
      error: new Error("fail"),
    } as any);
    render(<LayoutPage />, { wrapper });
    expect(screen.getByText(/エラー/)).toBeInTheDocument();
  });

  test("未選択時は選択プロンプトが表示される", () => {
    setupLists();
    render(<LayoutPage />, { wrapper });
    expect(
      screen.getByText("左の一覧からレイアウトを選択してください"),
    ).toBeInTheDocument();
  });

  test("一覧からアイテムを選択すると編集エリアに表示される", async () => {
    setupLists();
    render(<LayoutPage />, { wrapper });

    await userEvent.click(screen.getByText("dungeon_a"));

    expect(screen.getByDisplayValue("dungeon_a")).toBeInTheDocument();
    expect(screen.getByDisplayValue("100")).toBeInTheDocument();
  });

  test("保存ボタンでupdateResourceが呼ばれる", async () => {
    setupLists();
    render(<LayoutPage />, { wrapper });

    await userEvent.click(screen.getByText("dungeon_a"));
    await userEvent.click(screen.getByText("保存"));

    expect(mockMutate).toHaveBeenCalledWith(
      expect.objectContaining({ index: 0 }),
      expect.any(Object),
    );
  });

  test("新規追加でcreateResourceが呼ばれる", async () => {
    setupLists();
    render(<LayoutPage />, { wrapper });

    await userEvent.click(screen.getByText("＋"));

    expect(mockCreateMutate).toHaveBeenCalledWith(
      expect.objectContaining({ name: "new_layout" }),
      expect.any(Object),
    );
  });

  test("削除ボタンを2回クリックでdeleteResourceが呼ばれる", async () => {
    setupLists();
    render(<LayoutPage />, { wrapper });

    const deleteButtons = screen.getAllByText("×");
    await userEvent.click(deleteButtons[0]!);
    expect(screen.getByText("本当に?")).toBeInTheDocument();

    await userEvent.click(screen.getByText("本当に?"));
    expect(mockDeleteMutate).toHaveBeenCalledWith(0, expect.any(Object));
  });

  test("レイアウト選択後にプレビューが表示される", async () => {
    setupLists();
    render(<LayoutPage />, { wrapper });

    await userEvent.click(screen.getByText("dungeon_a"));
    expect(screen.getByTestId("map-preview")).toBeInTheDocument();
  });

  test("nameフィールドを編集できる", async () => {
    setupLists();
    render(<LayoutPage />, { wrapper });

    await userEvent.click(screen.getByText("dungeon_a"));
    const nameInput = screen.getByDisplayValue("dungeon_a");
    await userEvent.clear(nameInput);
    await userEvent.type(nameInput, "renamed");

    expect(screen.getByDisplayValue("renamed")).toBeInTheDocument();
  });

  test("件数バッジが表示される", () => {
    setupLists();
    render(<LayoutPage />, { wrapper });
    expect(screen.getByText("2")).toBeInTheDocument();
  });
});
