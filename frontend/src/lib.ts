import { useQuery } from "@tanstack/react-query";
import {
  ApiError,
  type AssetStatus,
  get,
  type ItemKind,
  type ItemSummary,
  type Location,
  type Meta,
  type Owned,
  type Stock,
  type User,
} from "./api";
import { dict } from "./i18n";
import { serverText } from "./i18n/server";
import { stem, translations } from "./search";

export const useMe = () => useQuery({ queryKey: ["me"], queryFn: () => get<User>("/api/me"), retry: false });

export const useMeta = () =>
  useQuery({
    queryKey: ["meta"],
    queryFn: () => get<Meta>("/api/meta"),
    staleTime: Number.POSITIVE_INFINITY,
  });

export const useLocations = () =>
  useQuery({ queryKey: ["locations"], queryFn: () => get<Location[]>("/api/locations") });

/** Every item with its totals; shared by the dashboard, the item list and scanning. */
export const allItemsQuery = {
  queryKey: ["items", "all"],
  queryFn: () => get<ItemSummary[]>("/api/items?limit=5000"),
};

export const useAllItems = () => useQuery(allItemsQuery);

export const isUnitKind = (k: ItemKind) => k === "tool" || k === "flight_hw";

/** Counted stock below its minimum (uncounted "lots" never are). */
export function isLow(it: ItemSummary): boolean {
  return (
    !isUnitKind(it.kind) &&
    it.uncounted_lots === 0 &&
    it.min_stock != null &&
    it.total_quantity < it.min_stock
  );
}

export function greeting(name: string): string {
  const g = dict().greeting;
  const h = new Date().getHours();
  const part = h < 5 ? g.evening : h < 12 ? g.morning : h < 19 ? g.afternoon : g.evening;
  return `${part}, ${name.split(" ")[0]}`;
}

export const itemKinds: ItemKind[] = ["consumable", "tool", "hazmat", "flight_hw"];

export const kindLabel = (k: ItemKind): string => dict().kind[k];

export const statusLabel = (s: AssetStatus): string => dict().status[s];

export const code = {
  location: (id: number) => `L${id}`,
  item: (id: number) => `I${id}`,
  stock: (id: number) => `S${id}`,
  asset: (id: number) => `A${id}`,
};

export function qty(n: number, unit = ""): string {
  const s = Number.isInteger(n) ? String(n) : n.toFixed(3).replace(/\.?0+$/, "");
  return unit && unit !== "pcs" ? `${s} ${unit}` : s;
}

/** A stock lot's quantity: "12 m", "≈10", or "Lots" when nobody counted. */
export function stockQty(s: Pick<Stock, "quantity" | "count_state">, unit = ""): string {
  if (s.count_state === "uncounted") return dict().common.lots;
  return (s.count_state === "approx" ? "≈" : "") + qty(s.quantity, unit);
}

/** Total of an item over all its lots, e.g. "40", "≈40", "40 + lots", "Lots". */
export function itemTotal(
  it: Pick<ItemSummary, "total_quantity" | "uncounted_lots" | "approx_lots" | "unit">,
) {
  const counted = (it.approx_lots > 0 ? "≈" : "") + qty(it.total_quantity, it.unit);
  if (it.uncounted_lots === 0) return counted;
  const c = dict().common;
  return it.total_quantity > 0 ? `${counted} ${c.plusLots}` : c.lots;
}

/** "Resistor 10k · 0603" */
export function itemTitle(it: { name: string; package: string }): string {
  return it.package ? `${it.name} · ${it.package}` : it.name;
}

/** A calendar date ("2026-12-31") in the chosen language, e.g. "31 Dec 2026". */
export function day(iso: string | null | undefined): string {
  if (!iso) return "";
  const d = new Date(`${iso.slice(0, 10)}T00:00:00Z`);
  if (Number.isNaN(d.getTime())) return iso.slice(0, 10);
  return d.toLocaleDateString(dict().locale, { dateStyle: "medium", timeZone: "UTC" });
}

export function when(iso: string | null | undefined): string {
  if (!iso) return "";
  const t = dict();
  const d = new Date(iso);
  const diff = (Date.now() - d.getTime()) / 1000;
  if (diff >= 0 && diff < 60) return t.time.justNow;
  if (diff >= 0 && diff < 3600) return t.time.minAgo(Math.floor(diff / 60));
  if (diff >= 0 && diff < 86400) return t.time.hAgo(Math.floor(diff / 3600));
  return d.toLocaleString(t.locale, { dateStyle: "medium", timeStyle: "short" });
}

/** "INVICTUS · Propulsion", "ICARUS", or "Team". */
export function ownerLabel(meta: Meta | undefined, o: Owned): string {
  return forLabel(meta, o.owner_project_id, o.owner_department_id) || dict().common.team;
}

export function forLabel(
  meta: Meta | undefined,
  projectId: number | null,
  departmentId: number | null,
): string {
  if (!meta) return "";
  if (departmentId) {
    const d = meta.departments.find((x) => x.id === departmentId);
    const p = d && meta.projects.find((x) => x.id === d.project_id);
    return d && p ? `${p.code} · ${d.name}` : "";
  }
  if (projectId) return meta.projects.find((x) => x.id === projectId)?.code ?? "";
  return "";
}

/** Full path of each location, e.g. "Room › Shelf A › Bin 3". */
export function locationPaths(locs: Location[] | undefined): Map<number, string> {
  const byId = new Map((locs ?? []).map((l) => [l.id, l]));
  const out = new Map<number, string>();
  const path = (l: Location, depth = 0): string => {
    const cached = out.get(l.id);
    if (cached) return cached;
    const parent = l.parent_id ? byId.get(l.parent_id) : undefined;
    const p = parent && depth < 32 ? `${path(parent, depth + 1)} › ${l.name}` : l.name;
    out.set(l.id, p);
    return p;
  };
  for (const l of byId.values()) path(l);
  return out;
}

// The "for what" choice on take/borrow is remembered per device.
const PURPOSE_KEY = "inventarius.purpose";

export interface PurposeChoice {
  project_id: number | null;
  department_id: number | null;
}

export function loadPurpose(me: User | undefined): PurposeChoice {
  try {
    const saved = localStorage.getItem(PURPOSE_KEY);
    if (saved) return JSON.parse(saved) as PurposeChoice;
  } catch {
    // storage unavailable
  }
  return { project_id: null, department_id: me?.department_ids[0] ?? null };
}

export function savePurpose(p: PurposeChoice) {
  try {
    localStorage.setItem(PURPOSE_KEY, JSON.stringify(p));
  } catch {
    // storage unavailable
  }
}

/** An error for the UI; server messages are translated when known. */
export function errorMessage(e: unknown): string {
  return e instanceof ApiError ? serverText(e.message) : e instanceof Error ? e.message : String(e);
}

/** Lower case without accents, so "gaveta" finds "Gavetão" and "armario" finds "Armário". */
export function fold(s: string): string {
  return s.normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase();
}

/**
 * Locations matching what was typed, best first. Every word must appear
 * somewhere in the path ("lab gav 3" finds "Laboratório › Armário A › Gaveta 3"),
 * in singular or plural, Portuguese or English ("drawer" finds "Gaveta 3"),
 * and a label code such as "L12" finds that location. A typed word that is a
 * whole word of the path counts more than the start of one, which counts more
 * than a piece from the middle; matches in the place's own name count extra.
 */
export function matchLocations(paths: Map<number, string>, query: string): number[] {
  const typed = fold(query)
    .split(/[\s›>/]+/)
    .filter(Boolean);
  if (typed.length === 0) return [];
  const byCode = typed.length === 1 ? /^l(\d+)$/.exec(typed[0]) : null;
  const wordsOf = (s: string) =>
    fold(s)
      .split(/[^\p{L}\p{N}]+/u)
      .filter(Boolean);
  const hits: { id: number; score: number; path: string }[] = [];
  for (const [id, path] of paths) {
    if (byCode && id === Number(byCode[1])) {
      hits.push({ id, score: Number.POSITIVE_INFINITY, path });
      continue;
    }
    const all = wordsOf(path);
    const own = wordsOf(path.slice(path.lastIndexOf("›") + 1));
    let score = 0;
    for (const w of typed) {
      const sw = stem(w);
      const others = translations(w);
      const grade = (ws: string[]) => {
        const g = ws.some((x) => x === w || stem(x) === sw)
          ? 3
          : ws.some((x) => x.startsWith(w))
            ? 2
            : ws.some((x) => x.includes(w))
              ? 1
              : 0;
        if (g >= 2) return g;
        // The same word in the other language ("drawer" for "Gaveta").
        for (const o of others) {
          const so = stem(o);
          if (ws.some((x) => x === o || stem(x) === so)) return 2.5;
          if (o.length >= 4 && ws.some((x) => x.startsWith(o))) return Math.max(g, 1.5);
        }
        return g;
      };
      const s = grade(all);
      if (s === 0) {
        score = -1;
        break;
      }
      score += s + (grade(own) > 0 ? 1 : 0);
    }
    if (score >= 0) hits.push({ id, score, path });
  }
  return hits
    .sort((a, b) => b.score - a.score || a.path.localeCompare(b.path, undefined, { numeric: true }))
    .map((h) => h.id);
}

// The locations picked last on this device, newest first.
const RECENT_KEY = "inventarius.recentLocations";
const RECENT_MAX = 6;

export function recentLocations(): number[] {
  try {
    const v = JSON.parse(localStorage.getItem(RECENT_KEY) ?? "[]");
    return Array.isArray(v) ? v.filter((x): x is number => typeof x === "number") : [];
  } catch {
    return [];
  }
}

export function rememberLocation(id: number) {
  try {
    const next = [id, ...recentLocations().filter((x) => x !== id)].slice(0, RECENT_MAX);
    localStorage.setItem(RECENT_KEY, JSON.stringify(next));
  } catch {
    // storage unavailable
  }
}
