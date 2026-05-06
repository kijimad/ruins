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
      expect.objectContaining({ name: "新規" }),
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

  test("削除ボタンは2回クリックで実行される", async () => {
    setupList([
      { name: "剣", value: 100 },
      { name: "盾", value: 200 },
    ]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });

    // 1回目: 確認状態になる
    const deleteButtons = screen.getAllByRole("button", { name: "×" });
    await userEvent.click(deleteButtons[0]!);
    expect(screen.getByText("本当に?")).toBeInTheDocument();
    expect(mockDeleteMutate).not.toHaveBeenCalled();

    // 2回目: 削除が実行される
    await userEvent.click(screen.getByText("本当に?"));
    expect(mockDeleteMutate).toHaveBeenCalledWith(
      0,
      expect.objectContaining({ onSuccess: expect.any(Function) }),
    );
  });

  test("ローディング中は読み込みメッセージが表示される", () => {
    setupList([], {
      data: undefined,
      isLoading: true,
      error: null,
    });

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    expect(screen.getByText("読み込み中...")).toBeInTheDocument();
  });

  test("エラー時はエラーメッセージが表示される", () => {
    setupList([], {
      data: undefined,
      isLoading: false,
      error: new Error("Network error"),
    });

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    expect(screen.getByText(/エラー:/)).toBeInTheDocument();
    expect(screen.getByText(/Network error/)).toBeInTheDocument();
  });

  test("選択前は案内メッセージが表示される", () => {
    setupList([{ name: "剣" }]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    expect(screen.getByText("左の一覧から項目を選択してください")).toBeInTheDocument();
  });

  test("一覧にアイテム件数バッジが表示される", () => {
    setupList([{ name: "剣" }, { name: "盾" }, { name: "杖" }]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    expect(screen.getByText("3")).toBeInTheDocument();
  });

  test("保存ボタンでupdateが呼ばれる", async () => {
    setupList([{ name: "剣", value: 100 }]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("剣"));
    await userEvent.click(screen.getByText("保存"));

    expect(mockMutate).toHaveBeenCalledWith(
      { index: 0, data: { name: "剣", value: 100 } },
      expect.objectContaining({ onError: expect.any(Function) }),
    );
  });

  test("フィールド編集後に保存すると変更内容が送信される", async () => {
    setupList([{ name: "剣", value: 100 }]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("剣"));

    const nameInput = screen.getByDisplayValue("剣");
    await userEvent.clear(nameInput);
    await userEvent.type(nameInput, "魔剣");

    await userEvent.click(screen.getByText("保存"));

    expect(mockMutate).toHaveBeenCalledWith(
      { index: 0, data: expect.objectContaining({ name: "魔剣" }) },
      expect.any(Object),
    );
  });

  test("数値フィールドの編集が正しく反映される", async () => {
    setupList([{ name: "剣", value: 100 }]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("剣"));

    const valueInput = screen.getByDisplayValue("100");
    await userEvent.clear(valueInput);
    await userEvent.type(valueInput, "250");

    await userEvent.click(screen.getByText("保存"));

    expect(mockMutate).toHaveBeenCalledWith(
      { index: 0, data: expect.objectContaining({ value: 250 }) },
      expect.any(Object),
    );
  });

  test("セクション削除ボタンでセクションが消える", async () => {
    setupList([{ name: "剣", weapon: {}, melee: { accuracy: 100, damage: 5 } }]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("剣"));

    expect(screen.getByText("melee")).toBeInTheDocument();

    // melee セクションの削除ボタンをクリック
    const deleteButtons = screen.getAllByRole("button", { name: "削除" });
    const meleeDelete = deleteButtons.find((btn) => {
      const legend = btn.closest("legend");
      return legend?.textContent?.includes("melee");
    });
    expect(meleeDelete).toBeDefined();
    await userEvent.click(meleeDelete!);

    // melee が消えてセクション追加ボタンに表示される
    await waitFor(() => {
      const addButtons = screen.getAllByRole("button").filter(
        (btn) => btn.textContent === "+ melee",
      );
      expect(addButtons.length).toBeGreaterThan(0);
    });
  });

  test("boolean フィールドがスイッチとして表示される", async () => {
    setupList([{ name: "テスト", active: true }]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("テスト"));

    expect(screen.getByText("active")).toBeInTheDocument();
    expect(screen.getByRole("checkbox")).toBeChecked();
  });

  test("ネストしたオブジェクトフィールドが表示される", async () => {
    setupList([{
      name: "回復薬",
      consumable: { targetGroup: "ALLY", targetNum: "SINGLE" },
    }]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("回復薬"));

    expect(screen.getByText("consumable")).toBeInTheDocument();
    expect(screen.getByDisplayValue("ALLY")).toBeInTheDocument();
    expect(screen.getByDisplayValue("SINGLE")).toBeInTheDocument();
  });

  test("削除成功後に選択状態がリセットされる", async () => {
    setupList([{ name: "剣" }, { name: "盾" }]);
    mockDeleteMutate.mockImplementation(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (_index: number, opts?: { onSuccess?: () => void }) => {
        opts?.onSuccess?.();
      },
    );

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });

    // 剣を選択
    await userEvent.click(screen.getByText("剣"));
    expect(screen.getByDisplayValue("剣")).toBeInTheDocument();

    // 剣を削除（2回クリック）
    const deleteButtons = screen.getAllByRole("button", { name: "×" });
    await userEvent.click(deleteButtons[0]!);
    await userEvent.click(screen.getByText("本当に?"));

    // 選択がリセットされ、案内メッセージが表示される
    await waitFor(() => {
      expect(screen.getByText("左の一覧から項目を選択してください")).toBeInTheDocument();
    });
  });

  test("保存エラー時にエラーメッセージが表示される", async () => {
    setupList([{ name: "剣" }]);
    mockMutate.mockImplementation(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (_data: unknown, opts?: { onError?: (err: any) => void }) => {
        opts?.onError?.(new Error("保存に失敗しました"));
      },
    );

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("剣"));
    await userEvent.click(screen.getByText("保存"));

    await waitFor(() => {
      expect(screen.getByText(/保存に失敗しました/)).toBeInTheDocument();
    });
  });

  test("プリミティブ配列に要素を追加できる", async () => {
    setupList([{ name: "スライム", animKeys: ["slime_0", "slime_1"] }]);

    render(<ResourcePage resource="members" label="メンバー" />, { wrapper });
    await userEvent.click(screen.getByText("スライム"));

    // 2つの入力が表示される
    expect(screen.getByDisplayValue("slime_0")).toBeInTheDocument();
    expect(screen.getByDisplayValue("slime_1")).toBeInTheDocument();

    // 追加ボタンをクリック
    const addButton = screen.getAllByRole("button").find(
      (btn) => btn.textContent === "＋ 追加",
    );
    expect(addButton).toBeDefined();
    await userEvent.click(addButton!);

    // 保存して新しい要素が含まれることを確認
    await userEvent.click(screen.getByText("保存"));
    expect(mockMutate).toHaveBeenCalledWith(
      { index: 0, data: expect.objectContaining({ animKeys: ["slime_0", "slime_1", ""] }) },
      expect.any(Object),
    );
  });

  test("プリミティブ配列から要素を削除できる", async () => {
    setupList([{ name: "スライム", animKeys: ["slime_0", "slime_1"] }]);

    render(<ResourcePage resource="members" label="メンバー" />, { wrapper });
    await userEvent.click(screen.getByText("スライム"));

    // animKeys の最初の×ボタンをクリック（削除ボタンは一覧の×とは別）
    const slime0Input = screen.getByDisplayValue("slime_0");
    const removeButton = slime0Input.parentElement?.querySelector("button");
    expect(removeButton).toBeDefined();
    await userEvent.click(removeButton!);

    await userEvent.click(screen.getByText("保存"));
    expect(mockMutate).toHaveBeenCalledWith(
      { index: 0, data: expect.objectContaining({ animKeys: ["slime_1"] }) },
      expect.any(Object),
    );
  });

  test("オブジェクト配列に要素を追加できる", async () => {
    setupList([{ name: "鉄の剣レシピ", inputs: [{ name: "鉄", amount: 4 }] }]);

    render(<ResourcePage resource="recipes" label="レシピ" />, { wrapper });
    await userEvent.click(screen.getByText("鉄の剣レシピ"));

    // name はSearchableSelectで表示されるのでテキストとして確認する
    expect(screen.getByText("鉄")).toBeInTheDocument();
    expect(screen.getByDisplayValue("4")).toBeInTheDocument();

    const addButton = screen.getAllByRole("button").find(
      (btn) => btn.textContent === "＋ 追加",
    );
    await userEvent.click(addButton!);

    await userEvent.click(screen.getByText("保存"));
    expect(mockMutate).toHaveBeenCalledWith(
      {
        index: 0,
        data: expect.objectContaining({
          inputs: [{ name: "鉄", amount: 4 }, { name: "", amount: 1 }],
        }),
      },
      expect.any(Object),
    );
  });

  test("オブジェクト配列から要素を削除できる", async () => {
    setupList([{
      name: "鉄の剣レシピ",
      inputs: [{ name: "鉄", amount: 4 }, { name: "木材", amount: 2 }],
    }]);

    render(<ResourcePage resource="recipes" label="レシピ" />, { wrapper });
    await userEvent.click(screen.getByText("鉄の剣レシピ"));

    // #0 の×ボタンをクリック
    const itemHeaders = screen.getAllByText(/^#\d+$/);
    const firstItemRemove = itemHeaders[0]!.parentElement?.querySelector("button");
    expect(firstItemRemove).toBeDefined();
    await userEvent.click(firstItemRemove!);

    await userEvent.click(screen.getByText("保存"));
    expect(mockMutate).toHaveBeenCalledWith(
      {
        index: 0,
        data: expect.objectContaining({
          inputs: [{ name: "木材", amount: 2 }],
        }),
      },
      expect.any(Object),
    );
  });

  test("空の配列に追加ボタンが表示される", async () => {
    setupList([{ name: "空レシピ", inputs: [] }]);

    render(<ResourcePage resource="recipes" label="レシピ" />, { wrapper });
    await userEvent.click(screen.getByText("空レシピ"));

    const addButton = screen.getAllByRole("button").find(
      (btn) => btn.textContent === "＋ 追加",
    );
    expect(addButton).toBeDefined();
  });

  test("選択式フィールドがセレクトボックスで表示される", async () => {
    setupList([{
      name: "剣",
      melee: { attackCategory: "SWORD", element: "NONE", targetGroup: "ENEMY", targetNum: "SINGLE" },
    }]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("剣"));

    // attackCategory がセレクトボックスになっている
    const selects = screen.getAllByRole("combobox");
    const values = selects.map((s) => (s as HTMLSelectElement).value);
    expect(values).toContain("SWORD");
    expect(values).toContain("NONE");
    expect(values).toContain("ENEMY");
    expect(values).toContain("SINGLE");
  });

  test("選択式フィールドの値を変更して保存できる", async () => {
    setupList([{
      name: "剣",
      melee: { attackCategory: "SWORD", element: "NONE", targetGroup: "ENEMY", targetNum: "SINGLE" },
    }]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("剣"));

    // element を CHILL に変更
    const selects = screen.getAllByRole("combobox");
    const elementSelect = selects.find((s) => (s as HTMLSelectElement).value === "NONE");
    expect(elementSelect).toBeDefined();
    await userEvent.selectOptions(elementSelect!, "CHILL");

    await userEvent.click(screen.getByText("保存"));
    expect(mockMutate).toHaveBeenCalledWith(
      {
        index: 0,
        data: expect.objectContaining({
          melee: expect.objectContaining({ element: "CHILL" }),
        }),
      },
      expect.any(Object),
    );
  });

  test("spriteSheetName がセレクトボックスで表示される", async () => {
    setupList([{ name: "剣", spriteSheetName: "field", spriteKey: "wooden_sword" }]);

    render(<ResourcePage resource="items" label="アイテム" />, { wrapper });
    await userEvent.click(screen.getByText("剣"));

    const selects = screen.getAllByRole("combobox");
    const sheetSelect = selects.find((s) => (s as HTMLSelectElement).value === "field");
    expect(sheetSelect).toBeDefined();
    // field, tile, bg の選択肢がある
    const options = Array.from((sheetSelect as HTMLSelectElement).options).map((o) => o.value);
    expect(options).toContain("field");
    expect(options).toContain("tile");
    expect(options).toContain("bg");
  });

  test("数値enumフィールドがラベル付きセレクトボックスで表示される", async () => {
    setupList([{
      name: "草原タイル",
      foliage: -1, shelter: 0, water: 0,
      spriteRender: { depth: 0, spriteKey: "dirt", spriteSheetName: "tile" },
    }]);

    render(<ResourcePage resource="tiles" label="タイル" />, { wrapper });
    await userEvent.click(screen.getByText("草原タイル"));

    const selects = screen.getAllByRole("combobox");
    // foliage=-1 のセレクトを探す
    const foliageSelect = selects.find((s) => (s as HTMLSelectElement).value === "-1");
    expect(foliageSelect).toBeDefined();
    const foliageOptions = Array.from((foliageSelect as HTMLSelectElement).options).map((o) => o.text);
    expect(foliageOptions).toContain("なし (0)");
    expect(foliageOptions).toContain("草原 (-1)");
    expect(foliageOptions).toContain("森 (-3)");

    // depth のセレクトも存在する
    const depthSelect = selects.find((s) => {
      const opts = Array.from((s as HTMLSelectElement).options).map((o) => o.text);
      return opts.some((t) => t.includes("Floor"));
    });
    expect(depthSelect).toBeDefined();
  });

  test("数値enumフィールドの値を変更して保存できる", async () => {
    setupList([{
      name: "草原タイル",
      foliage: 0, shelter: 0, water: 0,
      spriteRender: { depth: 0, spriteKey: "dirt", spriteSheetName: "tile" },
    }]);

    render(<ResourcePage resource="tiles" label="タイル" />, { wrapper });
    await userEvent.click(screen.getByText("草原タイル"));

    // foliage を森(-3)に変更
    const selects = screen.getAllByRole("combobox");
    const foliageSelect = selects.find((s) => {
      const opts = Array.from((s as HTMLSelectElement).options).map((o) => o.text);
      return opts.some((t) => t.includes("草原"));
    });
    expect(foliageSelect).toBeDefined();
    await userEvent.selectOptions(foliageSelect!, "-3");

    await userEvent.click(screen.getByText("保存"));
    expect(mockMutate).toHaveBeenCalledWith(
      {
        index: 0,
        data: expect.objectContaining({ foliage: -3 }),
      },
      expect.any(Object),
    );
  });

  test("allowEmpty な選択式フィールドに空選択肢がある", async () => {
    setupList([{ name: "スライム", factionType: "" }]);

    render(<ResourcePage resource="members" label="メンバー" />, { wrapper });
    await userEvent.click(screen.getByText("スライム"));

    const selects = screen.getAllByRole("combobox");
    const factionSelect = selects.find((s) => (s as HTMLSelectElement).value === "");
    expect(factionSelect).toBeDefined();
    const options = Array.from((factionSelect as HTMLSelectElement).options).map((o) => o.value);
    expect(options).toContain("");
    expect(options).toContain("FactionNeutral");
  });
});
