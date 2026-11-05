/**
 * Free-text matching done on the device, the same way the server's search
 * does it (backend/internal/search): without accents, in plural or
 * singular, in Portuguese or English ("parafuso" finds "Screw M3"), with
 * "ohm" for Ω and "u" for µ, units run together ("10kohm") and any word cut
 * short ("cond").
 */
import type { ItemSummary } from "./api";
import synonymGroups from "./synonyms.txt?raw";

export interface Field {
  text: string;
  weight: number;
}

const SYMBOLS: Record<string, string> = { ω: " ohm ", μ: "u", ohms: "ohm" };

/** Lower case, no accents, unit symbols spelled out. */
export function normalize(s: string): string {
  const folded = s
    .normalize("NFKD")
    .replace(/\p{Mn}/gu, "")
    .normalize("NFC")
    .toLowerCase();
  return folded.replace(/ω|μ|ohms/g, (m) => SYMBOLS[m]);
}

const isWordChar = (c: string) => /[\p{L}\p{N}]/u.test(c);
const isDigit = (c: string | undefined) => !!c && /\p{N}/u.test(c);

/** Normalized words; decimal points and commas inside numbers are kept ("0,47" -> "0.47"). */
export function tokens(s: string): string[] {
  const cs = [...normalize(s)];
  const out: string[] = [];
  let b = "";
  cs.forEach((c, i) => {
    if (isWordChar(c)) b += c;
    else if ((c === "." || c === ",") && b && isDigit(cs[i - 1]) && isDigit(cs[i + 1])) b += ".";
    else if (b) {
      out.push(b);
      b = "";
    }
  });
  if (b) out.push(b);
  return out;
}

/** Portuguese and English plurals to the singular ("condensadores", "botoes", "screws"). */
export function stem(w: string): string {
  const n = w.length;
  if (n <= 3 || !/\p{L}/u.test(w[n - 1])) return w;
  if (w.endsWith("oes")) return `${w.slice(0, -3)}ao`;
  if (w.endsWith("ais")) return `${w.slice(0, -3)}al`;
  if (["res", "zes", "les", "ses", "xes", "ches"].some((x) => w.endsWith(x))) return w.slice(0, -2);
  if (w.endsWith("ss")) return w;
  if (w.endsWith("s")) return w.slice(0, -1);
  return w;
}

/** A word's stem -> the other words of its synonym groups (synonyms.txt). */
const synonyms = (() => {
  const m = new Map<string, string[]>();
  for (const line of synonymGroups.split("\n")) {
    if (line.trim().startsWith("#")) continue;
    const words = line.split(/\s+/).filter(Boolean);
    for (const w of words) {
      const k = stem(w);
      const list = m.get(k) ?? [];
      list.push(...words.filter((o) => o !== w));
      m.set(k, list);
    }
  }
  return m;
})();

/** The other-language words for w, each once; 4+ letters also expand while typing ("capac"). */
export function translations(w: string): string[] {
  const ws = stem(w);
  const found: string[][] = [];
  if (ws.length < 4) found.push(synonyms.get(ws) ?? []);
  else for (const [k, v] of synonyms) if (k.startsWith(ws)) found.push(v);
  const out = new Set(found.flat());
  out.delete(w);
  return [...out].sort();
}

/** One way a typed word can match: all parts must be found. */
interface Alt {
  parts: string[];
  factor: number; // < 1 for looser forms (split units, translations)
  whole: boolean; // must be a whole word, not just its start
}

/** A parsed search: every typed word must match some word of a document. */
export type Query = Alt[][];

export function parseQuery(q: string): Query {
  return tokens(q).map((w) => {
    const alts: Alt[] = [{ parts: [w], factor: 1, whole: false }];
    for (let i = 1; i < w.length; i++) {
      const left = w.slice(0, i);
      const right = w.slice(i);
      if (/\p{N}/u.test(left) && !/\p{N}/u.test(right))
        alts.push({ parts: [left, right], factor: 0.9, whole: false });
    }
    // A short translation ("cap", "pot", "fan") must be a whole word.
    for (const t of translations(w)) alts.push({ parts: [t], factor: 0.95, whole: t.length < 4 });
    return alts;
  });
}

interface Token {
  word: string;
  stem: string;
}

/** A document prepared for matching. */
export interface Doc {
  fields: Token[][];
  weights: number[];
}

export function makeDoc(...fields: Field[]): Doc {
  return {
    fields: fields.map((f) => tokens(f.text).map((word) => ({ word, stem: stem(word) }))),
    weights: fields.map((f) => f.weight),
  };
}

function matchPart(d: Doc, p: string, whole: boolean): number {
  const ps = stem(p);
  let best = 0;
  d.fields.forEach((ts, i) => {
    ts.forEach((t, j) => {
      let s = 0;
      if (t.word === p || t.stem === ps) s = 3;
      else if (!whole && (t.word.startsWith(p) || t.stem.startsWith(ps))) s = 1;
      if (s > 0 && j === 0) s += 0.5;
      best = Math.max(best, s * d.weights[i]);
    });
  });
  return best;
}

/** How well the document matches; 0 when it doesn't. */
export function score(q: Query, d: Doc): number {
  let total = 0;
  for (const alts of q) {
    let best = 0;
    for (const a of alts) {
      let s = 0;
      for (const p of a.parts) {
        const ps = matchPart(d, p, a.whole);
        if (ps === 0) {
          s = 0;
          break;
        }
        s += ps;
      }
      best = Math.max(best, s * a.factor);
    }
    if (best === 0) return 0;
    total += best;
  }
  return total;
}

/** The things matching q, best first; ties keep their order. Empty q: none. */
export function rank<T>(things: readonly T[], q: string, fields: (t: T) => Field[]): T[] {
  const query = parseQuery(q);
  if (query.length === 0) return [];
  return things
    .map((t, i) => ({ t, i, s: score(query, makeDoc(...fields(t))) }))
    .filter((h) => h.s > 0)
    .sort((a, b) => b.s - a.s || a.i - b.i)
    .map((h) => h.t);
}

/** What an item is searched by on the device, weighted like the server does. */
export function itemFields(it: ItemSummary): Field[] {
  return [
    { text: it.name, weight: 3 },
    { text: `${it.package} ${it.mpn} ${it.label_text}`, weight: 2 },
    { text: it.manufacturer, weight: 1 },
    { text: it.description, weight: 0.5 },
  ];
}

/** Items matching q, best first. */
export function rankItems<T extends ItemSummary>(items: readonly T[], q: string): T[] {
  return rank(items, q, itemFields);
}
