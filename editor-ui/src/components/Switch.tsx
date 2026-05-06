import { Switch as ChakraSwitch } from "@chakra-ui/react";

interface SwitchProps {
  checked: boolean;
  onCheckedChange: (details: { checked: boolean }) => void;
  size?: string;
}

export function Switch({ checked, onCheckedChange, size }: SwitchProps) {
  return (
    <ChakraSwitch.Root
      checked={checked}
      onCheckedChange={onCheckedChange}
      size={size as "sm" | "md" | "lg"}
    >
      <ChakraSwitch.HiddenInput />
      <ChakraSwitch.Control>
        <ChakraSwitch.Thumb />
      </ChakraSwitch.Control>
    </ChakraSwitch.Root>
  );
}
