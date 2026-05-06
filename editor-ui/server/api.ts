import fs from "node:fs";
import path from "node:path";
import type { Plugin, ViteDevServer } from "vite";
import * as TOML from "smol-toml";

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
    sortAll(this.raws);
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
}

export function editorApiPlugin(options: ApiPluginOptions): Plugin {
  const rawTomlPath = path.resolve(options.rawTomlPath);
  const palettesDir = path.resolve(options.palettesDir);

  return {
    name: "editor-api",
    configureServer(server: ViteDevServer) {
      const rawStore = new RawStore(rawTomlPath);
      const paletteStore = new PaletteStore(palettesDir);

      server.middlewares.use(async (req, res, next) => {
        const url = req.url ?? "";
        if (!url.startsWith("/api/v1/")) return next();

        const apiPath = url.slice("/api/v1/".length);
        const method = req.method ?? "GET";

        res.setHeader("Content-Type", "application/json");

        try {
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
