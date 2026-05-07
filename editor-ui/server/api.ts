import fs from "node:fs";
import path from "node:path";
import type { Plugin, ViteDevServer } from "vite";
import * as TOML from "smol-toml";

// Aseprite JSON のフレーム定義
interface AsepriteFrame {
  filename: string;
  frame: { x: number; y: number; w: number; h: number };
}

interface AsepriteJson {
  frames: AsepriteFrame[];
  meta: { image: string; size: { w: number; h: number } };
}

// スプライトキー情報（APIレスポンス用）
interface SpriteInfo {
  key: string;
  x: number;
  y: number;
  w: number;
  h: number;
}

interface SpriteSheetInfo {
  name: string;
  image: string;
  sheetWidth: number;
  sheetHeight: number;
  sprites: SpriteInfo[];
}

// スプライトシートの名前→Aseprite JSONパスの対応をraw.tomlから取得し、
// スプライトキー一覧を返す
class SpriteCache {
  private cache = new Map<string, SpriteSheetInfo>();
  constructor(private assetsDir: string) {}

  get(sheetName: string, sheetPath: string): SpriteSheetInfo | undefined {
    if (this.cache.has(sheetName)) return this.cache.get(sheetName);
    const jsonPath = path.join(this.assetsDir, sheetPath);
    if (!fs.existsSync(jsonPath)) return undefined;
    const data: AsepriteJson = JSON.parse(fs.readFileSync(jsonPath, "utf-8"));
    const info: SpriteSheetInfo = {
      name: sheetName,
      image: `/sprites/${data.meta.image}`,
      sheetWidth: data.meta.size.w,
      sheetHeight: data.meta.size.h,
      sprites: data.frames.map((f) => ({
        key: f.filename.replace(/_$/, ""),
        x: f.frame.x,
        y: f.frame.y,
        w: f.frame.w,
        h: f.frame.h,
      })),
    };
    this.cache.set(sheetName, info);
    return info;
  }
}

// raw.toml のルート構造。キーは camelCase
interface Raws {
  items?: unknown[];
  recipes?: unknown[];
  members?: unknown[];
  commandTables?: unknown[];
  dropTables?: unknown[];
  itemTables?: unknown[];
  enemyTables?: unknown[];
  spriteSheets?: unknown[];
  tiles?: unknown[];
  props?: unknown[];
  professions?: unknown[];
}

// パレット TOML 構造
interface PaletteFile {
  palette: {
    id: string;
    description: string;
    terrain: Record<string, string>;
    props?: Record<string, { id: string; tile: string }>;
    npcs?: Record<string, { id: string; tile: string }>;
  };
}

// レイアウト TOML 構造
interface LayoutChunk {
  name: string;
  weight: number;
  palettes: string[];
  map: string;
  Size: { W: number; H: number };
  spawn_points: { x: number; y: number }[];
  placements: { id: string; chunks: string[] }[];
}

interface LayoutFile {
  chunk: LayoutChunk[];
}

// ソートキー生成（Go の itemSortKey と同じロジック）
function itemSortKey(item: Record<string, unknown>): string {
  const flags: [boolean, string][] = [
    [item["weapon"] != null, "A"],
    [item["melee"] != null, "B"],
    [item["fire"] != null, "C"],
    [item["wearable"] != null, "D"],
    [item["consumable"] != null, "E"],
    [item["ammo"] != null, "F"],
    [item["book"] != null, "G"],
  ];
  const key = flags
    .filter(([present]) => present)
    .map(([, code]) => code)
    .join("");
  return key || "Z";
}

function sortItems(items: Record<string, unknown>[]): void {
  items.sort((a, b) => {
    const ka = itemSortKey(a);
    const kb = itemSortKey(b);
    if (ka !== kb) return ka < kb ? -1 : 1;
    return String(a["name"] ?? "") < String(b["name"] ?? "") ? -1 : 1;
  });
}

function sortByName(arr: Record<string, unknown>[], key: string): void {
  arr.sort((a, b) => {
    const va = String(a[key] ?? "");
    const vb = String(b[key] ?? "");
    return va < vb ? -1 : va > vb ? 1 : 0;
  });
}

function sortAll(raws: Raws): void {
  if (raws.items) sortItems(raws.items as Record<string, unknown>[]);
  if (raws.members)
    sortByName(raws.members as Record<string, unknown>[], "name");
  if (raws.recipes)
    sortByName(raws.recipes as Record<string, unknown>[], "name");
  if (raws.commandTables)
    sortByName(raws.commandTables as Record<string, unknown>[], "name");
  if (raws.dropTables)
    sortByName(raws.dropTables as Record<string, unknown>[], "name");
  if (raws.itemTables)
    sortByName(raws.itemTables as Record<string, unknown>[], "name");
  if (raws.enemyTables)
    sortByName(raws.enemyTables as Record<string, unknown>[], "name");
  if (raws.spriteSheets)
    sortByName(raws.spriteSheets as Record<string, unknown>[], "name");
  if (raws.tiles) sortByName(raws.tiles as Record<string, unknown>[], "name");
  if (raws.props) sortByName(raws.props as Record<string, unknown>[], "name");
  if (raws.professions)
    sortByName(raws.professions as Record<string, unknown>[], "id");
}

// raw.toml の読み書き
class RawStore {
  private raws: Raws;
  constructor(private filePath: string) {
    this.raws = this.load();
  }

  private load(): Raws {
    const content = fs.readFileSync(this.filePath, "utf-8");
    const raws = TOML.parse(content) as Raws;
    sortAll(raws);
    return raws;
  }

  private save(): void {
    const content = TOML.stringify(this.raws as unknown as Record<string, unknown>);
    fs.writeFileSync(this.filePath, content, "utf-8");
  }

  getSlice(key: keyof Raws): unknown[] {
    return (this.raws[key] as unknown[]) ?? [];
  }

  getAt(key: keyof Raws, index: number): unknown {
    const slice = this.getSlice(key);
    if (index < 0 || index >= slice.length) return undefined;
    return slice[index];
  }

  addTo(key: keyof Raws, item: unknown): number {
    if (!this.raws[key]) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (this.raws as any)[key] = [];
    }
    (this.raws[key] as unknown[]).push(item);
    this.save();
    // ソート後のインデックスを返す
    return (this.raws[key] as unknown[]).indexOf(item);
  }

  updateAt(key: keyof Raws, index: number, item: unknown): boolean {
    const slice = this.raws[key] as unknown[] | undefined;
    if (!slice || index < 0 || index >= slice.length) return false;
    slice[index] = item;
    this.save();
    return true;
  }

  deleteAt(key: keyof Raws, index: number): boolean {
    const slice = this.raws[key] as unknown[] | undefined;
    if (!slice || index < 0 || index >= slice.length) return false;
    slice.splice(index, 1);
    this.save();
    return true;
  }
}

// パレット TOML の読み書き
class PaletteStore {
  constructor(private dir: string) {}

  private list(): PaletteFile["palette"][] {
    const entries = fs.readdirSync(this.dir);
    const palettes: PaletteFile["palette"][] = [];
    for (const entry of entries) {
      if (!entry.endsWith(".toml")) continue;
      const content = fs.readFileSync(path.join(this.dir, entry), "utf-8");
      const parsed = TOML.parse(content) as unknown as PaletteFile;
      if (parsed.palette) palettes.push(parsed.palette);
    }
    palettes.sort((a, b) => (a.id < b.id ? -1 : a.id > b.id ? 1 : 0));
    return palettes;
  }

  getAll(): PaletteFile["palette"][] {
    return this.list();
  }

  get(id: string): PaletteFile["palette"] | undefined {
    return this.list().find((p) => p.id === id);
  }

  save(palette: PaletteFile["palette"]): void {
    const safe = path.basename(palette.id);
    if (safe !== palette.id || safe === "." || safe === "..") {
      throw new Error(`不正なパレットID: ${palette.id}`);
    }
    const filePath = path.join(this.dir, `${safe}.toml`);
    const file: PaletteFile = { palette };
    const content = TOML.stringify(file as unknown as Record<string, unknown>);
    fs.writeFileSync(filePath, content, "utf-8");
  }

  delete(id: string): boolean {
    const safe = path.basename(id);
    if (safe !== id || safe === "." || safe === "..") return false;
    const filePath = path.join(this.dir, `${safe}.toml`);
    if (!fs.existsSync(filePath)) return false;
    fs.unlinkSync(filePath);
    return true;
  }
}

// レイアウト TOML の読み書き
class LayoutStore {
  constructor(private dir: string) {}

  // ファイル名（拡張子なし）を返す
  private fileNames(): string[] {
    return fs.readdirSync(this.dir)
      .filter((f) => f.endsWith(".toml"))
      .map((f) => f.replace(/\.toml$/, ""))
      .sort();
  }

  private readFile(fileName: string): LayoutChunk | undefined {
    const filePath = path.join(this.dir, `${fileName}.toml`);
    if (!fs.existsSync(filePath)) return undefined;
    const content = fs.readFileSync(filePath, "utf-8");
    const parsed = TOML.parse(content) as unknown as LayoutFile;
    if (!parsed.chunk || parsed.chunk.length === 0) return undefined;
    return parsed.chunk[0];
  }

  getAll(): LayoutChunk[] {
    return this.fileNames()
      .map((f) => this.readFile(f))
      .filter((c): c is LayoutChunk => c !== undefined);
  }

  getAt(index: number): LayoutChunk | undefined {
    const names = this.fileNames();
    if (index < 0 || index >= names.length) return undefined;
    return this.readFile(names[index]!);
  }

  save(chunk: LayoutChunk): void {
    // ファイル名はチャンク名から生成する
    const fileName = chunk.name.replace(/[^a-zA-Z0-9_-]/g, "_");
    const filePath = path.join(this.dir, `${fileName}.toml`);
    const file: LayoutFile = { chunk: [chunk] };
    const content = TOML.stringify(file as unknown as Record<string, unknown>);
    fs.writeFileSync(filePath, content, "utf-8");
  }

  updateAt(index: number, chunk: LayoutChunk): boolean {
    const names = this.fileNames();
    if (index < 0 || index >= names.length) return false;
    const oldName = names[index]!;
    const newFileName = chunk.name.replace(/[^a-zA-Z0-9_-]/g, "_");
    // ファイル名が変わった場合は旧ファイルを削除する
    if (newFileName !== oldName) {
      const oldPath = path.join(this.dir, `${oldName}.toml`);
      if (fs.existsSync(oldPath)) fs.unlinkSync(oldPath);
    }
    this.save(chunk);
    return true;
  }

  add(chunk: LayoutChunk): number {
    this.save(chunk);
    // ソート後のインデックスを返す
    const names = this.fileNames();
    const fileName = chunk.name.replace(/[^a-zA-Z0-9_-]/g, "_");
    return names.indexOf(fileName);
  }

  deleteAt(index: number): boolean {
    const names = this.fileNames();
    if (index < 0 || index >= names.length) return false;
    const filePath = path.join(this.dir, `${names[index]!}.toml`);
    if (!fs.existsSync(filePath)) return false;
    fs.unlinkSync(filePath);
    return true;
  }
}

// リソース種別 → Raws のキーと識別フィールドのマッピング
const RESOURCE_MAP: Record<
  string,
  { key: keyof Raws; idField: string; hasGet: boolean }
> = {
  items: { key: "items", idField: "name", hasGet: true },
  members: { key: "members", idField: "name", hasGet: true },
  recipes: { key: "recipes", idField: "name", hasGet: true },
  tiles: { key: "tiles", idField: "name", hasGet: true },
  props: { key: "props", idField: "name", hasGet: true },
  professions: { key: "professions", idField: "id", hasGet: false },
  "command-tables": { key: "commandTables", idField: "name", hasGet: false },
  "drop-tables": { key: "dropTables", idField: "name", hasGet: false },
  "item-tables": { key: "itemTables", idField: "name", hasGet: false },
  "enemy-tables": { key: "enemyTables", idField: "name", hasGet: false },
  "sprite-sheets": { key: "spriteSheets", idField: "name", hasGet: false },
};

// リクエストボディを読み取る
function readBody(req: { on: (event: string, cb: (chunk: Buffer) => void) => void }): Promise<string> {
  return new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    req.on("data", (chunk: Buffer) => chunks.push(chunk));
    req.on("end", () => resolve(Buffer.concat(chunks).toString()));
    req.on("error", reject);
  });
}

interface ApiPluginOptions {
  rawTomlPath: string;
  palettesDir: string;
  layoutsDir: string;
  chunkDirs: string[];
  assetsDir: string;
}

export function editorApiPlugin(options: ApiPluginOptions): Plugin {
  const rawTomlPath = path.resolve(options.rawTomlPath);
  const palettesDir = path.resolve(options.palettesDir);
  const layoutsDir = path.resolve(options.layoutsDir);
  const chunkDirs = options.chunkDirs.map((d) => path.resolve(d));
  const assetsDir = path.resolve(options.assetsDir);
  const spritesDir = path.join(assetsDir, "file/textures/dist");

  return {
    name: "editor-api",
    configureServer(server: ViteDevServer) {
      const rawStore = new RawStore(rawTomlPath);
      const paletteStore = new PaletteStore(palettesDir);
      const layoutStore = new LayoutStore(layoutsDir);
      const spriteCache = new SpriteCache(assetsDir);

      // スプライトシート画像の静的配信
      server.middlewares.use((req, res, next) => {
        const url = req.url ?? "";
        if (!url.startsWith("/sprites/")) return next();
        const fileName = path.basename(url.slice("/sprites/".length));
        const filePath = path.join(spritesDir, fileName);
        if (!fs.existsSync(filePath)) {
          res.statusCode = 404;
          res.end("Not found");
          return;
        }
        res.setHeader("Content-Type", "image/png");
        res.setHeader("Cache-Control", "public, max-age=3600");
        fs.createReadStream(filePath).pipe(res);
      });

      server.middlewares.use(async (req, res, next) => {
        const url = req.url ?? "";
        if (!url.startsWith("/api/v1/")) return next();

        const apiPath = url.slice("/api/v1/".length);
        const method = req.method ?? "GET";

        res.setHeader("Content-Type", "application/json");

        try {
          // スプライトキー一覧 API
          const spriteMatch = apiPath.match(/^sprites\/([a-zA-Z0-9_-]+)$/);
          if (spriteMatch && method === "GET") {
            const sheetName = spriteMatch[1]!;
            const sheets = rawStore.getSlice("spriteSheets") as { name: string; path: string }[];
            const sheet = sheets.find((s) => s.name === sheetName);
            if (!sheet) {
              res.statusCode = 404;
              res.end(JSON.stringify({ message: `Sprite sheet not found: ${sheetName}` }));
              return;
            }
            const info = spriteCache.get(sheetName, sheet.path);
            if (!info) {
              res.statusCode = 404;
              res.end(JSON.stringify({ message: `Sprite JSON not found: ${sheet.path}` }));
              return;
            }
            res.end(JSON.stringify(info));
            return;
          }

          // パレット API
          if (apiPath.startsWith("palettes")) {
            await handlePalettes(
              paletteStore,
              apiPath,
              method,
              req as Parameters<typeof readBody>[0],
              res,
            );
            return;
          }

          // レイアウト API
          if (apiPath.startsWith("layouts")) {
            await handleLayouts(
              layoutStore,
              paletteStore,
              chunkDirs,
              apiPath,
              method,
              req as Parameters<typeof readBody>[0],
              res,
            );
            return;
          }

          // 汎用 CRUD API
          await handleResource(
            rawStore,
            apiPath,
            method,
            req as Parameters<typeof readBody>[0],
            res,
          );
        } catch (e) {
          res.statusCode = 500;
          res.end(
            JSON.stringify({ message: e instanceof Error ? e.message : String(e) }),
          );
        }
      });
    },
  };
}

type Res = {
  statusCode: number;
  setHeader: (key: string, value: string) => void;
  end: (body: string) => void;
};

async function handleResource(
  store: RawStore,
  apiPath: string,
  method: string,
  req: Parameters<typeof readBody>[0],
  res: Res,
): Promise<void> {
  // "items" or "items/3"
  const match = apiPath.match(/^([a-z-]+)(?:\/(\d+))?$/);
  if (!match) {
    res.statusCode = 404;
    res.end(JSON.stringify({ message: "Not found" }));
    return;
  }

  const resourceName = match[1];
  const indexStr = match[2];
  const mapping = RESOURCE_MAP[resourceName];

  if (!mapping) {
    res.statusCode = 404;
    res.end(JSON.stringify({ message: `Unknown resource: ${resourceName}` }));
    return;
  }

  const { key } = mapping;

  if (method === "GET" && indexStr === undefined) {
    // List
    const data = store.getSlice(key);
    res.end(JSON.stringify({ data, totalCount: data.length }));
    return;
  }

  if (method === "GET" && indexStr !== undefined) {
    // Get
    const index = parseInt(indexStr, 10);
    const item = store.getAt(key, index);
    if (item === undefined) {
      res.statusCode = 404;
      res.end(JSON.stringify({ message: `Index out of range: ${index}` }));
      return;
    }
    res.end(JSON.stringify(item));
    return;
  }

  if (method === "POST") {
    // Create
    const body = JSON.parse(await readBody(req));
    const newIndex = store.addTo(key, body);
    res.statusCode = 201;
    res.end(JSON.stringify({ index: newIndex, data: body }));
    return;
  }

  if (method === "PUT" && indexStr !== undefined) {
    // Update
    const index = parseInt(indexStr, 10);
    const body = JSON.parse(await readBody(req));
    if (!store.updateAt(key, index, body)) {
      res.statusCode = 400;
      res.end(JSON.stringify({ message: `Index out of range: ${index}` }));
      return;
    }
    res.end(JSON.stringify(body));
    return;
  }

  if (method === "DELETE" && indexStr !== undefined) {
    // Delete
    const index = parseInt(indexStr, 10);
    if (!store.deleteAt(key, index)) {
      res.statusCode = 400;
      res.end(JSON.stringify({ message: `Index out of range: ${index}` }));
      return;
    }
    res.statusCode = 204;
    res.end("");
    return;
  }

  res.statusCode = 405;
  res.end(JSON.stringify({ message: "Method not allowed" }));
}

async function handlePalettes(
  store: PaletteStore,
  apiPath: string,
  method: string,
  req: Parameters<typeof readBody>[0],
  res: Res,
): Promise<void> {
  // "palettes" or "palettes/some-id"
  const match = apiPath.match(/^palettes(?:\/(.+))?$/);
  if (!match) {
    res.statusCode = 404;
    res.end(JSON.stringify({ message: "Not found" }));
    return;
  }
  const id = match[1];

  if (method === "GET" && id === undefined) {
    const data = store.getAll();
    res.end(JSON.stringify({ data, totalCount: data.length }));
    return;
  }

  if (method === "GET" && id !== undefined) {
    const palette = store.get(id);
    if (!palette) {
      res.statusCode = 404;
      res.end(JSON.stringify({ message: `Palette not found: ${id}` }));
      return;
    }
    res.end(JSON.stringify(palette));
    return;
  }

  if (method === "POST") {
    const body = JSON.parse(await readBody(req));
    store.save(body);
    res.statusCode = 201;
    res.end(JSON.stringify(body));
    return;
  }

  if (method === "PUT" && id !== undefined) {
    const body = JSON.parse(await readBody(req));
    body.id = id;
    store.save(body);
    res.end(JSON.stringify(body));
    return;
  }

  if (method === "DELETE" && id !== undefined) {
    if (!store.delete(id)) {
      res.statusCode = 400;
      res.end(JSON.stringify({ message: `Failed to delete: ${id}` }));
      return;
    }
    res.statusCode = 204;
    res.end("");
    return;
  }

  res.statusCode = 405;
  res.end(JSON.stringify({ message: "Method not allowed" }));
}

// マップの1セルを表す解決済みオブジェクト（Go側の MapCell と同じ構造）
interface MapCell {
  terrain: string;
  prop: string;
  npc: string;
}

// パレットを使ってマップ文字列をセル配列に解決する（Go側の ResolveMapCells と同じロジック）
function resolveMapCells(
  mapStr: string,
  palette: PaletteFile["palette"] | null,
): MapCell[][] {
  const lines = mapStr.trim().split("\n");
  return lines.map((line) =>
    [...line].map((ch) => {
      const cell: MapCell = { terrain: "", prop: "", npc: "" };
      if (palette) {
        if (palette.terrain[ch]) cell.terrain = palette.terrain[ch];
        if (palette.props?.[ch]) {
          cell.prop = palette.props[ch].id;
          if (!cell.terrain) cell.terrain = palette.props[ch].tile;
        }
        if (palette.npcs?.[ch]) {
          cell.npc = palette.npcs[ch].id;
          if (!cell.terrain) cell.terrain = palette.npcs[ch].tile;
        }
      }
      if (!cell.terrain && !cell.prop && !cell.npc) cell.terrain = ch;
      return cell;
    }),
  );
}

// パレットをマージする。後のパレットが優先
function mergePalettes(
  palettes: PaletteFile["palette"][],
): PaletteFile["palette"] {
  const merged: PaletteFile["palette"] = {
    id: "merged",
    description: "",
    terrain: {},
    props: {},
    npcs: {},
  };
  for (const pal of palettes) {
    Object.assign(merged.terrain, pal.terrain);
    if (pal.props) Object.assign(merged.props!, pal.props);
    if (pal.npcs) Object.assign(merged.npcs!, pal.npcs);
  }
  return merged;
}

// プレースホルダ領域の位置とサイズ
interface PlaceholderRegion {
  x: number;
  y: number;
  width: number;
  height: number;
}

// 識別子(ID)に対応するプレースホルダ領域を全て検出する
function findPlaceholderRegionsByID(
  lines: string[],
  id: string,
): PlaceholderRegion[] {
  const idChar = id;
  const placeholder = "@";

  // IDの全出現位置を見つける
  const positions: [number, number][] = [];
  for (let y = 0; y < lines.length; y++) {
    for (let x = 0; x < lines[y]!.length; x++) {
      if (lines[y]![x] === idChar) positions.push([x, y]);
    }
  }
  if (positions.length === 0) return [];

  const isPlaceholder = (ch: string) => ch === placeholder || ch === idChar;

  const regions: PlaceholderRegion[] = [];
  for (const [idX, idY] of positions) {
    // 左端を探す
    let startX = idX;
    while (startX > 0 && isPlaceholder(lines[idY]![startX - 1]!)) startX--;

    // 幅を計算
    let width = 0;
    for (let x = startX; x < lines[idY]!.length && isPlaceholder(lines[idY]![x]!); x++) width++;

    // 上端を探す
    let startY = idY;
    while (startY > 0) {
      let allMatch = true;
      for (let x = startX; x < startX + width; x++) {
        if (x >= lines[startY - 1]!.length || !isPlaceholder(lines[startY - 1]![x]!)) {
          allMatch = false;
          break;
        }
      }
      if (allMatch) startY--;
      else break;
    }

    // 高さを計算
    let height = 0;
    for (let y = startY; y < lines.length && startX < lines[y]!.length && isPlaceholder(lines[y]![startX]!); y++) height++;

    regions.push({ x: startX, y: startY, width, height });
  }

  return regions;
}

// 複数ディレクトリからすべてのチャンクを読み込む
function loadAllChunks(dirs: string[]): LayoutChunk[] {
  const chunks: LayoutChunk[] = [];
  for (const dir of dirs) {
    const resolved = path.resolve(dir);
    if (!fs.existsSync(resolved)) continue;
    for (const entry of fs.readdirSync(resolved)) {
      if (!entry.endsWith(".toml")) continue;
      const content = fs.readFileSync(path.join(resolved, entry), "utf-8");
      const parsed = TOML.parse(content) as unknown as LayoutFile;
      if (parsed.chunk) {
        for (const c of parsed.chunk) chunks.push(c);
      }
    }
  }
  return chunks;
}

// レイアウトのplacementsを再帰的に展開し、解決済みセル配列を返す
function resolveLayoutCells(
  chunk: LayoutChunk,
  allChunks: LayoutChunk[],
  paletteStore: PaletteStore,
  depth: number = 0,
): MapCell[][] {
  if (depth > 10) throw new Error(`チャンク展開の深度が制限を超えました`);

  // このチャンクのパレットをマージして解決する
  const palettes = chunk.palettes
    .map((id) => paletteStore.get(id))
    .filter((p): p is PaletteFile["palette"] => p !== undefined);
  const merged = palettes.length > 0 ? mergePalettes(palettes) : null;
  const cells = resolveMapCells(chunk.map, merged);

  if (!chunk.placements || chunk.placements.length === 0) return cells;

  const lines = chunk.map.trim().split("\n");

  for (const placement of chunk.placements) {
    if (!placement.id) continue;

    const regions = findPlaceholderRegionsByID(lines, placement.id);
    for (const region of regions) {
      // チャンク候補から最初のものを選択する（プレビューなのでランダムは不要）
      const childName = placement.chunks[0];
      if (!childName) continue;
      const childChunk = allChunks.find((c) => c.name === childName);
      if (!childChunk) continue;

      // サイズチェック
      if (region.width !== childChunk.Size.W || region.height !== childChunk.Size.H) continue;

      // 子チャンクを再帰的に展開する
      const childCells = resolveLayoutCells(childChunk, allChunks, paletteStore, depth + 1);

      // 子のセルを親のセルにオーバーレイする
      for (let cy = 0; cy < childChunk.Size.H; cy++) {
        for (let cx = 0; cx < childChunk.Size.W; cx++) {
          const ty = region.y + cy;
          const tx = region.x + cx;
          if (ty < cells.length && tx < (cells[ty]?.length ?? 0)) {
            cells[ty]![tx] = childCells[cy]![cx]!;
          }
        }
      }
    }
  }

  return cells;
}

async function handleLayouts(
  store: LayoutStore,
  paletteStore: PaletteStore,
  chunkDirs: string[],
  apiPath: string,
  method: string,
  req: Parameters<typeof readBody>[0],
  res: Res,
): Promise<void> {
  // resolved エンドポイント
  const resolvedMatch = apiPath.match(/^layouts\/(\d+)\/resolved$/);
  if (resolvedMatch && method === "GET") {
    const index = parseInt(resolvedMatch[1]!, 10);
    const chunk = store.getAt(index);
    if (!chunk) {
      res.statusCode = 404;
      res.end(JSON.stringify({ message: `Index out of range: ${index}` }));
      return;
    }
    const allChunks = loadAllChunks(chunkDirs);
    const cells = resolveLayoutCells(chunk, allChunks, paletteStore);
    res.end(JSON.stringify({ cells }));
    return;
  }

  const match = apiPath.match(/^layouts(?:\/(\d+))?$/);
  if (!match) {
    res.statusCode = 404;
    res.end(JSON.stringify({ message: "Not found" }));
    return;
  }
  const indexStr = match[1];

  if (method === "GET" && indexStr === undefined) {
    const data = store.getAll();
    res.end(JSON.stringify({ data, totalCount: data.length }));
    return;
  }

  if (method === "GET" && indexStr !== undefined) {
    const index = parseInt(indexStr, 10);
    const chunk = store.getAt(index);
    if (!chunk) {
      res.statusCode = 404;
      res.end(JSON.stringify({ message: `Index out of range: ${index}` }));
      return;
    }
    res.end(JSON.stringify(chunk));
    return;
  }

  if (method === "POST") {
    const body = JSON.parse(await readBody(req));
    const newIndex = store.add(body);
    res.statusCode = 201;
    res.end(JSON.stringify({ index: newIndex, data: body }));
    return;
  }

  if (method === "PUT" && indexStr !== undefined) {
    const index = parseInt(indexStr, 10);
    const body = JSON.parse(await readBody(req));
    if (!store.updateAt(index, body)) {
      res.statusCode = 400;
      res.end(JSON.stringify({ message: `Index out of range: ${index}` }));
      return;
    }
    res.end(JSON.stringify(body));
    return;
  }

  if (method === "DELETE" && indexStr !== undefined) {
    const index = parseInt(indexStr, 10);
    if (!store.deleteAt(index)) {
      res.statusCode = 400;
      res.end(JSON.stringify({ message: `Index out of range: ${index}` }));
      return;
    }
    res.statusCode = 204;
    res.end("");
    return;
  }

  res.statusCode = 405;
  res.end(JSON.stringify({ message: "Method not allowed" }));
}
