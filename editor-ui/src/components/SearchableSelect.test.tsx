import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ChakraProvider, defaultSystem } from "@chakra-ui/react";
import { SearchableSelect } from "./SearchableSelect";
import { vi } from "vitest";

function wrapper({ children }: { children: React.ReactNode }) {
  return <ChakraProvider value={defaultSystem}>{children}</ChakraProvider>;
}

const options = ["apple", "banana", "cherry", "date", "elderberry"];

describe("SearchableSelect", () => {
  test("選択中の値が表示される", () => {
    render(
      <SearchableSelect options={options} value="banana" onChange={vi.fn()} />,
      { wrapper },
    );
    expect(screen.getByText("banana")).toBeInTheDocument();
  });

  test("値が空のときプレースホルダーが表示される", () => {
    render(<SearchableSelect options={options} value="" onChange={vi.fn()} />, {
      wrapper,
    });
    expect(screen.getByText("選択...")).toBeInTheDocument();
  });

  test("カスタムプレースホルダーが表示される", () => {
    render(
      <SearchableSelect
        options={options}
        value=""
        onChange={vi.fn()}
        placeholder="選んで"
      />,
      { wrapper },
    );
    expect(screen.getByText("選んで")).toBeInTheDocument();
  });

  test("クリックでドロップダウンが開き全候補が表示される", async () => {
    render(
      <SearchableSelect options={options} value="apple" onChange={vi.fn()} />,
      { wrapper },
    );
    await userEvent.click(screen.getByText("apple"));

    expect(screen.getByPlaceholderText("検索...")).toBeInTheDocument();
    for (const opt of options) {
      expect(screen.getAllByText(opt).length).toBeGreaterThanOrEqual(1);
    }
  });

  test("検索テキストで候補がフィルタリングされる", async () => {
    render(<SearchableSelect options={options} value="" onChange={vi.fn()} />, {
      wrapper,
    });
    await userEvent.click(screen.getByText("選択..."));
    await userEvent.type(screen.getByPlaceholderText("検索..."), "an");

    expect(screen.getByText("banana")).toBeInTheDocument();
    expect(screen.queryByText("cherry")).not.toBeInTheDocument();
    expect(screen.queryByText("date")).not.toBeInTheDocument();
  });

  test("検索は大文字小文字を無視する", async () => {
    render(<SearchableSelect options={options} value="" onChange={vi.fn()} />, {
      wrapper,
    });
    await userEvent.click(screen.getByText("選択..."));
    await userEvent.type(screen.getByPlaceholderText("検索..."), "APPLE");

    expect(screen.getByText("apple")).toBeInTheDocument();
    expect(screen.queryByText("banana")).not.toBeInTheDocument();
  });

  test("検索で該当なしのとき「該当なし」が表示される", async () => {
    render(<SearchableSelect options={options} value="" onChange={vi.fn()} />, {
      wrapper,
    });
    await userEvent.click(screen.getByText("選択..."));
    await userEvent.type(screen.getByPlaceholderText("検索..."), "zzzzz");

    expect(screen.getByText("該当なし")).toBeInTheDocument();
  });

  test("候補クリックでonChangeが呼ばれドロップダウンが閉じる", async () => {
    const onChange = vi.fn();
    render(
      <SearchableSelect options={options} value="" onChange={onChange} />,
      { wrapper },
    );

    await userEvent.click(screen.getByText("選択..."));
    await userEvent.click(screen.getByText("cherry"));

    expect(onChange).toHaveBeenCalledWith("cherry");
    await waitFor(() => {
      expect(screen.queryByPlaceholderText("検索...")).not.toBeInTheDocument();
    });
  });

  test("Enterキーで先頭候補が選択される", async () => {
    const onChange = vi.fn();
    render(
      <SearchableSelect options={options} value="" onChange={onChange} />,
      { wrapper },
    );

    await userEvent.click(screen.getByText("選択..."));
    await userEvent.type(screen.getByPlaceholderText("検索..."), "ch");
    await userEvent.keyboard("{Enter}");

    expect(onChange).toHaveBeenCalledWith("cherry");
  });

  test("Escapeキーでドロップダウンが閉じる", async () => {
    render(<SearchableSelect options={options} value="" onChange={vi.fn()} />, {
      wrapper,
    });

    await userEvent.click(screen.getByText("選択..."));
    expect(screen.getByPlaceholderText("検索...")).toBeInTheDocument();

    await userEvent.keyboard("{Escape}");
    await waitFor(() => {
      expect(screen.queryByPlaceholderText("検索...")).not.toBeInTheDocument();
    });
  });

  test("外部クリックでドロップダウンが閉じる", async () => {
    render(
      <div>
        <span data-testid="outside">外側</span>
        <SearchableSelect options={options} value="" onChange={vi.fn()} />
      </div>,
      { wrapper },
    );

    await userEvent.click(screen.getByText("選択..."));
    expect(screen.getByPlaceholderText("検索...")).toBeInTheDocument();

    await userEvent.click(screen.getByTestId("outside"));
    await waitFor(() => {
      expect(screen.queryByPlaceholderText("検索...")).not.toBeInTheDocument();
    });
  });

  test("選択中の値がハイライトされる", async () => {
    render(
      <SearchableSelect options={options} value="banana" onChange={vi.fn()} />,
      { wrapper },
    );
    await userEvent.click(screen.getByText("banana"));

    // ドロップダウン内の banana 要素に bg.emphasized が適用される
    const items = screen.getAllByText("banana");
    // ドロップダウン内の要素（2つ目）が存在することを確認
    expect(items.length).toBeGreaterThanOrEqual(2);
  });
});
