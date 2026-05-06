import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ChakraProvider, defaultSystem } from "@chakra-ui/react";
import { SpriteSelect } from "./SpriteSelect";
import { vi } from "vitest";
import type { SpriteSheetInfo } from "../hooks/useSprites";

const mockSheet: SpriteSheetInfo = {
  name: "field",
  image: "/sprites/single.png",
  sheetWidth: 608,
  sheetHeight: 576,
  sprites: [
    { key: "alarm_clock", x: 96, y: 544, w: 32, h: 32 },
    { key: "wooden_sword", x: 0, y: 0, w: 32, h: 32 },
    { key: "armor_item", x: 32, y: 0, w: 32, h: 32 },
    { key: "player_0", x: 64, y: 0, w: 32, h: 32 },
    { key: "slime_0", x: 96, y: 0, w: 32, h: 32 },
  ],
};

vi.mock("../hooks/useSprites", () => ({
  useSpriteSheet: vi.fn(),
}));

import { useSpriteSheet } from "../hooks/useSprites";
const mockedUseSpriteSheet = vi.mocked(useSpriteSheet);

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <ChakraProvider value={defaultSystem}>
      <QueryClientProvider client={qc}>{children}</QueryClientProvider>
    </ChakraProvider>
  );
}

function setupSheet(sheet: SpriteSheetInfo | undefined) {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  mockedUseSpriteSheet.mockReturnValue({ data: sheet } as any);
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("SpriteSelect", () => {
  test("シートデータ未取得時はテキストinputにフォールバックする", () => {
    setupSheet(undefined);
    const onChange = vi.fn();
    render(<SpriteSelect sheetName="field" value="wooden_sword" onChange={onChange} />, { wrapper });

    const input = screen.getByPlaceholderText("spriteKey");
    expect(input).toBeInTheDocument();
    expect(input).toHaveValue("wooden_sword");
  });

  test("シートデータ取得後は現在の値がテキスト表示される", () => {
    setupSheet(mockSheet);
    const onChange = vi.fn();
    render(<SpriteSelect sheetName="field" value="wooden_sword" onChange={onChange} />, { wrapper });

    expect(screen.getByText("wooden_sword")).toBeInTheDocument();
  });

  test("クリックでドロップダウンが開き全候補が表示される", async () => {
    setupSheet(mockSheet);
    const onChange = vi.fn();
    render(<SpriteSelect sheetName="field" value="wooden_sword" onChange={onChange} />, { wrapper });

    await userEvent.click(screen.getByText("wooden_sword"));

    expect(screen.getByPlaceholderText("検索...")).toBeInTheDocument();
    for (const sprite of mockSheet.sprites) {
      // 選択中の値はヘッダーとドロップダウン両方に表示されうるため getAllByText を使う
      expect(screen.getAllByText(sprite.key).length).toBeGreaterThanOrEqual(1);
    }
  });

  test("検索テキストで候補がフィルタリングされる", async () => {
    setupSheet(mockSheet);
    const onChange = vi.fn();
    render(<SpriteSelect sheetName="field" value="" onChange={onChange} />, { wrapper });

    await userEvent.click(screen.getByText("(未選択)"));
    const searchInput = screen.getByPlaceholderText("検索...");
    await userEvent.type(searchInput, "sword");

    expect(screen.getByText("wooden_sword")).toBeInTheDocument();
    expect(screen.queryByText("alarm_clock")).not.toBeInTheDocument();
    expect(screen.queryByText("player_0")).not.toBeInTheDocument();
  });

  test("検索で該当なしのとき「該当なし」が表示される", async () => {
    setupSheet(mockSheet);
    const onChange = vi.fn();
    render(<SpriteSelect sheetName="field" value="" onChange={onChange} />, { wrapper });

    await userEvent.click(screen.getByText("(未選択)"));
    await userEvent.type(screen.getByPlaceholderText("検索..."), "zzzzz");

    expect(screen.getByText("該当なし")).toBeInTheDocument();
  });

  test("候補クリックでonChangeが呼ばれドロップダウンが閉じる", async () => {
    setupSheet(mockSheet);
    const onChange = vi.fn();
    render(<SpriteSelect sheetName="field" value="" onChange={onChange} />, { wrapper });

    await userEvent.click(screen.getByText("(未選択)"));
    await userEvent.click(screen.getByText("alarm_clock"));

    expect(onChange).toHaveBeenCalledWith("alarm_clock");
    await waitFor(() => {
      expect(screen.queryByPlaceholderText("検索...")).not.toBeInTheDocument();
    });
  });

  test("Enterキーで先頭候補が選択される", async () => {
    setupSheet(mockSheet);
    const onChange = vi.fn();
    render(<SpriteSelect sheetName="field" value="" onChange={onChange} />, { wrapper });

    await userEvent.click(screen.getByText("(未選択)"));
    const searchInput = screen.getByPlaceholderText("検索...");
    await userEvent.type(searchInput, "sword");
    await userEvent.keyboard("{Enter}");

    expect(onChange).toHaveBeenCalledWith("wooden_sword");
  });

  test("Escapeキーでドロップダウンが閉じる", async () => {
    setupSheet(mockSheet);
    const onChange = vi.fn();
    render(<SpriteSelect sheetName="field" value="" onChange={onChange} />, { wrapper });

    await userEvent.click(screen.getByText("(未選択)"));
    expect(screen.getByPlaceholderText("検索...")).toBeInTheDocument();

    await userEvent.keyboard("{Escape}");
    await waitFor(() => {
      expect(screen.queryByPlaceholderText("検索...")).not.toBeInTheDocument();
    });
  });

  test("外部クリックでドロップダウンが閉じる", async () => {
    setupSheet(mockSheet);
    const onChange = vi.fn();
    render(
      <div>
        <span data-testid="outside">外側</span>
        <SpriteSelect sheetName="field" value="" onChange={onChange} />
      </div>,
      { wrapper },
    );

    await userEvent.click(screen.getByText("(未選択)"));
    expect(screen.getByPlaceholderText("検索...")).toBeInTheDocument();

    await userEvent.click(screen.getByTestId("outside"));
    await waitFor(() => {
      expect(screen.queryByPlaceholderText("検索...")).not.toBeInTheDocument();
    });
  });

  test("検索は大文字小文字を無視する", async () => {
    setupSheet(mockSheet);
    const onChange = vi.fn();
    render(<SpriteSelect sheetName="field" value="" onChange={onChange} />, { wrapper });

    await userEvent.click(screen.getByText("(未選択)"));
    await userEvent.type(screen.getByPlaceholderText("検索..."), "SWORD");

    expect(screen.getByText("wooden_sword")).toBeInTheDocument();
  });

  test("値が空のとき「(未選択)」が表示される", () => {
    setupSheet(mockSheet);
    const onChange = vi.fn();
    render(<SpriteSelect sheetName="field" value="" onChange={onChange} />, { wrapper });

    expect(screen.getByText("(未選択)")).toBeInTheDocument();
  });
});
