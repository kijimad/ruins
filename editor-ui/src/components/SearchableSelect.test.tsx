import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ChakraProvider, defaultSystem } from "@chakra-ui/react";
import { SearchableSelect } from "./SearchableSelect";
import { vi } from "vitest";

function wrapper({ children }: { children: React.ReactNode }) {
  return <ChakraProvider value={defaultSystem}>{children}</ChakraProvider>;
}

const options = ["apple", "banana", "cherry", "date", "elderberry"];

// Comboboxのinput要素を取得するヘルパー
function getComboboxInput() {
  return screen.getByRole("combobox");
}

describe("SearchableSelect", () => {
  test("選択中の値がinputに表示される", () => {
    render(
      <SearchableSelect options={options} value="banana" onChange={vi.fn()} />,
      { wrapper },
    );
    expect(getComboboxInput()).toHaveValue("banana");
  });

  test("値が空のときプレースホルダーが表示される", () => {
    render(<SearchableSelect options={options} value="" onChange={vi.fn()} />, {
      wrapper,
    });
    expect(getComboboxInput()).toHaveAttribute("placeholder", "選択...");
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
    expect(getComboboxInput()).toHaveAttribute("placeholder", "選んで");
  });

  test("入力でドロップダウンが開き候補が表示される", async () => {
    render(<SearchableSelect options={options} value="" onChange={vi.fn()} />, {
      wrapper,
    });
    const input = getComboboxInput();
    await userEvent.click(input);
    await userEvent.type(input, "a");

    await waitFor(() => {
      expect(screen.getByText("apple")).toBeInTheDocument();
    });
  });

  test("検索テキストで候補がフィルタリングされる", async () => {
    render(<SearchableSelect options={options} value="" onChange={vi.fn()} />, {
      wrapper,
    });
    const input = getComboboxInput();
    await userEvent.type(input, "an");

    await waitFor(() => {
      expect(screen.getByText("banana")).toBeInTheDocument();
    });
  });

  test("候補クリックでonChangeが呼ばれる", async () => {
    const onChange = vi.fn();
    render(
      <SearchableSelect options={options} value="" onChange={onChange} />,
      { wrapper },
    );
    const input = getComboboxInput();
    await userEvent.click(input);
    await userEvent.type(input, "ch");

    await waitFor(() => {
      expect(screen.getByText("cherry")).toBeInTheDocument();
    });
    await userEvent.click(screen.getByText("cherry"));

    expect(onChange).toHaveBeenCalledWith("cherry");
  });
});
