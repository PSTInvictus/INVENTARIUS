import { useQuery } from "@tanstack/react-query";
import {
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

export const useMe = () => useQuery({ queryKey: ["me"], queryFn: () => get<User>("/api/me"), retry: false });

export const useMeta = () =>
  useQuery({
    queryKey: ["meta"],
    queryFn: () => get<Meta>("/api/meta"),
    staleTime: Number.POSITIVE_INFINITY,
  });

export const useLocations = () =>
  useQuery({ queryKey: ["locations"], queryFn: () => get<Location[]>("/api/locations") });

/** Every item with its totals; shared by the dashboard and the item list. */
export const useAllItems = () =>
  useQuery({ queryKey: ["items", "all"], queryFn: () => get<ItemSummary[]>("/api/items?limit=5000") });

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
  const h = new Date().getHours();
  const part = h < 5 ? "Good evening" : h < 12 ? "Good morning" : h < 19 ? "Good afternoon" : "Good evening";
  return `${part}, ${name.split(" ")[0]}`;
}

export const kindLabel: Record<ItemKind, string> = {
  consumable: "Consumable",
  tool: "Tool / equipment",
  hazmat: "Hazardous",
  flight_hw: "Flight hardware",
};

export const statusLabel: Record<AssetStatus, string> = {
  available: "Available",
  on_loan: "On loan",
  broken: "Broken",
  in_repair: "In repair",
  retired: "Retired",
};

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
  if (s.count_state === "uncounted") return "Lots";
  return (s.count_state === "approx" ? "≈" : "") + qty(s.quantity, unit);
}

/** Total of an item over all its lots, e.g. "40", "≈40", "40 + lots", "Lots". */
export function itemTotal(
  it: Pick<ItemSummary, "total_quantity" | "uncounted_lots" | "approx_lots" | "unit">,
) {
  const counted = (it.approx_lots > 0 ? "≈" : "") + qty(it.total_quantity, it.unit);
  if (it.uncounted_lots === 0) return counted;
  return it.total_quantity > 0 ? `${counted} + lots` : "Lots";
}

/** "Resistor 10k · 0603" */
export function itemTitle(it: { name: string; package: string }): string {
  return it.package ? `${it.name} · ${it.package}` : it.name;
}

export function day(iso: string | null | undefined): string {
  if (!iso) return "";
  return iso.slice(0, 10);
}

export function when(iso: string | null | undefined): string {
  if (!iso) return "";
  const d = new Date(iso);
  const diff = (Date.now() - d.getTime()) / 1000;
  if (diff >= 0 && diff < 60) return "just now";
  if (diff >= 0 && diff < 3600) return `${Math.floor(diff / 60)} min ago`;
  if (diff >= 0 && diff < 86400) return `${Math.floor(diff / 3600)} h ago`;
  return d.toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

/** "INVICTUS · Propulsion", "ICARUS", or "Team". */
export function ownerLabel(meta: Meta | undefined, o: Owned): string {
  return forLabel(meta, o.owner_project_id, o.owner_department_id) || "Team";
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

export function errorMessage(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}
