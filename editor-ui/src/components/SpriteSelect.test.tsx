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
  mockedUseSpriteSheet.mockReturnValue({ data: sheet } as any);
}

// Comboboxのinput要素を取得するヘルパー
function getComboboxInput() {
  return screen.getByRole("combobox");
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("SpriteSelect", () => {
  test("シートデータ未取得時はinputが無効化される", () => {
    setupSheet(undefined);
    render(
      <SpriteSelect sheetName="field" value="wooden_sword" onChange={vi.fn()} />,
      { wrapper },
    );

    const input = screen.getByPlaceholderText("読み込み中...");
    expect(input).toBeDisabled();
  });

  test("シートデータ取得後は現在の値がinputに表示される", () => {
    setupSheet(mockSheet);
    render(
      <SpriteSelect sheetName="field" value="wooden_sword" onChange={vi.fn()} />,
      { wrapper },
    );

    expect(getComboboxInput()).toHaveValue("wooden_sword");
  });

  test("入力で候補がフィルタリングされる", async () => {
    setupSheet(mockSheet);
    render(
      <SpriteSelect sheetName="field" value="" onChange={vi.fn()} />,
      { wrapper },
    );

    const input = getComboboxInput();
    await userEvent.type(input, "sword");

    await waitFor(() => {
      expect(screen.getByText("wooden_sword")).toBeInTheDocument();
    });
  });

  test("候補クリックでonChangeが呼ばれる", async () => {
    setupSheet(mockSheet);
    const onChange = vi.fn();
    render(
      <SpriteSelect sheetName="field" value="" onChange={onChange} />,
      { wrapper },
    );

    const input = getComboboxInput();
    await userEvent.click(input);
    await userEvent.type(input, "alarm");

    await waitFor(() => {
      expect(screen.getByText("alarm_clock")).toBeInTheDocument();
    });
    await userEvent.click(screen.getByText("alarm_clock"));

    expect(onChange).toHaveBeenCalledWith("alarm_clock");
  });

  test("値が空のときプレースホルダーが表示される", () => {
    setupSheet(mockSheet);
    render(
      <SpriteSelect sheetName="field" value="" onChange={vi.fn()} />,
      { wrapper },
    );

    expect(getComboboxInput()).toHaveAttribute("placeholder", "(未選択)");
  });
});
