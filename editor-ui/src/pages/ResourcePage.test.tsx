import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ChakraProvider, defaultSystem } from "@chakra-ui/react";
import { ResourcePage } from "./ResourcePage";
import { vi } from "vitest";

// useResource hooks をモック
const mockMutate = vi.fn();
const mockCreateMutate = vi.fn();
const mockDeleteMutate = vi.fn();

vi.mock("../hooks/useResource", () => ({
  useResourceList: vi.fn(),
  useResourceUpdate: vi.fn(() => ({ mutate: mockMutate, isPending: false })),
  useResourceCreate: vi.fn(() => ({ mutate: mockCreateMutate, isPending: false })),
  useResourceDelete: vi.fn(() => ({ mutate: mockDeleteMutate, isPending: false })),
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

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function setupList(items: Record<string, unknown>[], returnValue?: any) {
  const val = returnValue ?? {
    data: { data: items, totalCount: items.length },
    isLoading: false,
    error: null,
  };
  mockedUseResourceList.mockReturnValue(val);
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("ResourcePage", () => {
  test("一覧からアイテムを選択すると編集エリアに表示される", async () => {
    setupList([
      { name: "剣", value: 100 },
      { name: "盾", value: 200 },
    ]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });

    await userEvent.click(screen.getByText("剣"));
    expect(screen.getByDisplayValue("剣")).toBeInTheDocument();
    expect(screen.getByDisplayValue("100")).toBeInTheDocument();
  });

  test("新規追加でソート後のインデックスの項目が選択される", async () => {
    // ソート順: 剣, 新規, 盾 → 新規はindex=1
    setupList([{ name: "剣", value: 100 }, { name: "盾", value: 200 }]);

    mockCreateMutate.mockImplementation(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (_data: unknown, opts?: { onSuccess?: (result: any) => void }) => {
        // サーバーが返すソート後のリストとインデックス
        setupList([
          { name: "剣", value: 100 },
          { name: "新規" },
          { name: "盾", value: 200 },
        ]);
        // レスポンスにソート後のインデックスを含む
        opts?.onSuccess?.({ index: 1, data: { name: "新規" } });
      },
    );

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("＋"));

    expect(mockCreateMutate).toHaveBeenCalledWith(
      { name: "新規" },
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );

    // ソート後のインデックス1（「新規」）が選択され、編集フォームに表示される
    await waitFor(() => {
      expect(screen.getByDisplayValue("新規")).toBeInTheDocument();
    });
  });

  test("nameFieldを指定するとそのフィールドが一覧に表示される", () => {
    setupList([
      { id: "warrior", label: "戦士" },
      { id: "mage", label: "魔術師" },
    ]);

    render(<ResourcePage resource="professions" label="職業" nameField="id" />, { wrapper });

    expect(screen.getByText("warrior")).toBeInTheDocument();
    expect(screen.getByText("mage")).toBeInTheDocument();
  });

  test("セクション追加ボタンが未追加セクションのみ表示される", async () => {
    setupList([
      { name: "剣", weapon: {}, melee: { accuracy: 100 } },
    ]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("剣"));

    // weapon と melee は既にあるので追加ボタンに含まれない
    const addButtons = screen.getAllByRole("button").filter(
      (btn) => btn.textContent?.startsWith("+ "),
    );
    const addLabels = addButtons.map((btn) => btn.textContent?.replace("+ ", ""));
    expect(addLabels).not.toContain("weapon");
    expect(addLabels).not.toContain("melee");
    expect(addLabels).toContain("fire");
    expect(addLabels).toContain("wearable");
  });

  test("セクション追加するとデフォルト値付きでフィールドが表示される", async () => {
    setupList([{ name: "帽子" }]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("帽子"));

    // wearable セクション追加
    const addWearable = screen.getAllByRole("button").find(
      (btn) => btn.textContent === "+ wearable",
    );
    expect(addWearable).toBeDefined();
    await userEvent.click(addWearable!);

    // デフォルト値のフィールドが表示される
    await waitFor(() => {
      expect(screen.getByDisplayValue("TORSO")).toBeInTheDocument();
    });
    // defense=0 のフィールドも表示される
    expect(screen.getByText("defense")).toBeInTheDocument();
    expect(screen.getByText("insulationCold")).toBeInTheDocument();
  });

  test("items以外のリソースではセクション追加ボタンが表示されない", async () => {
    setupList([{ name: "レシピA" }]);

    render(<ResourcePage resource="recipes" label="レシピ" />, { wrapper });
    await userEvent.click(screen.getByText("レシピA"));

    const addButtons = screen.queryAllByRole("button").filter(
      (btn) => btn.textContent?.startsWith("+ "),
    );
    expect(addButtons).toHaveLength(0);
  });

  test("削除ボタンでconfirm後にdeleteが呼ばれる", async () => {
    setupList([
      { name: "剣", value: 100 },
      { name: "盾", value: 200 },
    ]);

    globalThis.confirm = vi.fn(() => true);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });

    // 「剣」の行の×ボタンをクリック
    const deleteButtons = screen.getAllByRole("button", { name: "×" });
    await userEvent.click(deleteButtons[0]!);

    expect(globalThis.confirm).toHaveBeenCalledWith("「剣」を削除しますか?");
    expect(mockDeleteMutate).toHaveBeenCalledWith(
      0,
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  test("削除でconfirmキャンセルするとdeleteは呼ばれない", async () => {
    setupList([{ name: "剣", value: 100 }]);

    globalThis.confirm = vi.fn(() => false);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });

    const deleteButtons = screen.getAllByRole("button", { name: "×" });
    await userEvent.click(deleteButtons[0]!);

    expect(globalThis.confirm).toHaveBeenCalled();
    expect(mockDeleteMutate).not.toHaveBeenCalled();
  });
});
