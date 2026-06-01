import { type ReactNode, useEffect, useMemo, useState } from "react";
import {
  ComboboxContent,
  ComboboxControl,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxItemText,
  ComboboxList,
  ComboboxPositioner,
  ComboboxRoot,
  ComboboxTrigger,
  Flex,
  createListCollection,
} from "@chakra-ui/react";

interface SearchableSelectProps {
  options: string[];
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  disabled?: boolean;
  // 選択中の値の横に表示するカスタム要素
  renderSelected?: (item: string) => ReactNode;
  // ドロップダウン内の各アイテムをカスタム描画する
  renderItem?: (item: string) => ReactNode;
}

export function SearchableSelect({
  options,
  value,
  onChange,
  placeholder = "選択...",
  disabled,
  renderSelected,
  renderItem,
}: SearchableSelectProps) {
  const collection = useMemo(
    () =>
      createListCollection({
        items: options,
        itemToString: (item) => item,
        itemToValue: (item) => item,
      }),
    [options],
  );

  const [inputValue, setInputValue] = useState(value);

  // 外部からvalue propが変わったときにinputValueを同期する
  useEffect(() => {
    setInputValue(value);
  }, [value]);

  // 入力値が選択値と同じ場合はフィルタリングしない（全件表示）
  const filterText = inputValue !== value ? inputValue.toLowerCase() : "";

  return (
    <ComboboxRoot
      collection={collection}
      value={value ? [value] : []}
      inputValue={inputValue}
      onInputValueChange={(details) => setInputValue(details.inputValue)}
      onValueChange={(details) => {
        const selected = details.value[0];
        if (selected !== undefined) {
          onChange(selected);
          setInputValue(selected);
        }
      }}
      disabled={disabled}
      allowCustomValue
      openOnClick
      size="sm"
      flex="1"
    >
      <ComboboxControl>
        <Flex align="center" flex="1">
          {renderSelected && value && renderSelected(value)}
          <ComboboxInput
            placeholder={placeholder}
            onBlur={() => {
              // 自由入力値もonChangeに反映する
              if (inputValue !== value) onChange(inputValue);
            }}
          />
        </Flex>
        <ComboboxTrigger />
      </ComboboxControl>
      <ComboboxPositioner>
        <ComboboxContent>
          <ComboboxEmpty>該当なし</ComboboxEmpty>
          <ComboboxList>
            {options.map((item) => {
              const matches =
                !filterText || item.toLowerCase().includes(filterText);
              return (
                <ComboboxItem key={item} item={item} hidden={!matches}>
                  {renderItem ? renderItem(item) : null}
                  <ComboboxItemText>{item}</ComboboxItemText>
                </ComboboxItem>
              );
            })}
          </ComboboxList>
        </ComboboxContent>
      </ComboboxPositioner>
    </ComboboxRoot>
  );
}
