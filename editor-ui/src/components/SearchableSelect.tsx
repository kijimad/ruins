import { useEffect, useMemo, useRef, useState } from "react";
import { Box, Flex, Input, Text } from "@chakra-ui/react";

interface SearchableSelectProps {
  options: string[];
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

export function SearchableSelect({ options, value, onChange, placeholder = "選択..." }: SearchableSelectProps) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  // 外部クリックで閉じる
  useEffect(() => {
    if (!open) return;
    function handleClick(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [open]);

  const filtered = useMemo(() => {
    if (!search) return options;
    const lower = search.toLowerCase();
    return options.filter((s) => s.toLowerCase().includes(lower));
  }, [options, search]);

  return (
    <Box ref={containerRef} position="relative" flex="1">
      <Flex
        align="center"
        gap="2"
        borderWidth="1px"
        borderRadius="md"
        px="2"
        py="1"
        cursor="pointer"
        onClick={() => {
          setOpen(!open);
          setSearch("");
          setTimeout(() => inputRef.current?.focus(), 0);
        }}
        _hover={{ borderColor: "border.emphasized" }}
      >
        <Text fontSize="sm" flex="1" truncate>
          {value || placeholder}
        </Text>
      </Flex>

      {open && (
        <Box
          position="absolute"
          top="100%"
          left="0"
          right="0"
          zIndex="10"
          bg="bg"
          borderWidth="1px"
          borderRadius="md"
          boxShadow="lg"
          mt="1"
          maxH="320px"
          overflow="hidden"
        >
          <Box p="1" borderBottom="1px solid" borderColor="border">
            <Input
              ref={inputRef}
              size="sm"
              placeholder="検索..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Escape") setOpen(false);
                if (e.key === "Enter" && filtered.length > 0) {
                  onChange(filtered[0]!);
                  setOpen(false);
                }
              }}
            />
          </Box>
          <Box overflowY="auto" maxH="272px">
            {filtered.length === 0 ? (
              <Text fontSize="sm" color="fg.muted" p="2">
                該当なし
              </Text>
            ) : (
              filtered.map((option) => (
                <Flex
                  key={option}
                  align="center"
                  px="2"
                  py="1"
                  cursor="pointer"
                  bg={option === value ? "bg.emphasized" : undefined}
                  _hover={{ bg: "bg.muted" }}
                  onClick={() => {
                    onChange(option);
                    setOpen(false);
                  }}
                >
                  <Text fontSize="sm" truncate>
                    {option}
                  </Text>
                </Flex>
              ))
            )}
          </Box>
        </Box>
      )}
    </Box>
  );
}
