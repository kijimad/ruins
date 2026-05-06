import { useQuery } from "@tanstack/react-query";
import axios from "axios";

export interface SpriteInfo {
  key: string;
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface SpriteSheetInfo {
  name: string;
  image: string;
  sheetWidth: number;
  sheetHeight: number;
  sprites: SpriteInfo[];
}

export function useSpriteSheet(sheetName: string | undefined) {
  return useQuery<SpriteSheetInfo>({
    queryKey: ["sprites", sheetName],
    queryFn: async () => {
      const res = await axios.get<SpriteSheetInfo>(`/api/v1/sprites/${sheetName}`);
      return res.data;
    },
    enabled: !!sheetName,
    staleTime: Infinity,
  });
}
