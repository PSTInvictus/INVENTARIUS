import { describe, expect, it } from "vitest";
import type { Location, Meta } from "./api";
import { fold, forLabel, locationPaths, matchLocations, qty } from "./lib";
import { expandCodes } from "./pages/PrintPage";

const loc = (id: number, name: string, parent_id: number | null = null): Location => ({
  id,
  name,
  parent_id,
  description: "",
  created_at: "",
  version: 1,
});

describe("expandCodes", () => {
  it("expands ranges and normalizes case", () => {
    expect(expandCodes("l4-l6, a9;S2")).toEqual(["L4", "L5", "L6", "A9", "S2"]);
    expect(expandCodes("L7-9")).toEqual(["L7", "L8", "L9"]);
  });
  it("rejects bad input", () => {
    expect(() => expandCodes("X1")).toThrow();
    expect(() => expandCodes("L9-L3")).toThrow();
    expect(() => expandCodes("L1-L500")).toThrow();
    expect(() => expandCodes("L0")).toThrow();
    expect(() => expandCodes("  ")).toThrow();
  });
});

describe("locationPaths", () => {
  it("joins ancestors", () => {
    const p = locationPaths([loc(3, "Bin 3", 2), loc(1, "Room"), loc(2, "Shelf A", 1)]);
    expect(p.get(3)).toBe("Room › Shelf A › Bin 3");
    expect(p.get(1)).toBe("Room");
  });
  it("survives a parent cycle", () => {
    const p = locationPaths([loc(1, "A", 2), loc(2, "B", 1)]);
    expect(p.get(1)).toContain("A");
  });
});

describe("formatting", () => {
  it("formats quantities", () => {
    expect(qty(3)).toBe("3");
    expect(qty(0.1 + 0.2, "m")).toBe("0.3 m");
    expect(qty(12, "pcs")).toBe("12");
  });
  it("names purposes", () => {
    const meta: Meta = {
      projects: [{ id: 1, code: "INVICTUS", name: "INVICTUS" }],
      departments: [{ id: 4, project_id: 1, name: "Propulsion" }],
      symbology: "datamatrix",
      label_url: "",
      part_lookup: false,
      part_sources: [],
    };
    expect(forLabel(meta, null, 4)).toBe("INVICTUS · Propulsion");
    expect(forLabel(meta, 1, null)).toBe("INVICTUS");
    expect(forLabel(meta, null, null)).toBe("");
  });
});

describe("matchLocations", () => {
  const paths = locationPaths([
    loc(1, "Laboratório"),
    loc(2, "Armário A", 1),
    loc(3, "Gaveta 3", 2),
    loc(4, "Gaveta 12", 2),
    loc(5, "Oficina"),
    loc(6, "Prateleira 3", 5),
  ]);
  it("needs every word somewhere in the path, ignoring case and accents", () => {
    expect(matchLocations(paths, "armario gaveta")).toEqual([3, 4]);
    expect(matchLocations(paths, "LAB 3")).toEqual([3]);
    expect(fold("Armário")).toBe("armario");
  });
  it("puts places whose own name matches first", () => {
    // "Oficina" itself before the shelf inside it.
    expect(matchLocations(paths, "oficina")).toEqual([5, 6]);
    // "3" is in the names of Gaveta 3 and Prateleira 3, and in no parent's.
    expect(matchLocations(paths, "3")).toEqual([3, 6]);
  });
  it("prefers whole words over the start of one", () => {
    // "1" is the whole name of Gaveta 1 but only starts Gaveta 12.
    const more = new Map([...paths, [7, "Laboratório › Armário A › Gaveta 1"]]);
    expect(matchLocations(more, "gav 1")).toEqual([7, 4]);
    // "a" is a whole word of Armário A, so its drawers come first.
    expect(matchLocations(paths, "a gav")).toEqual([3, 4]);
  });
  it("matches plurals and the other language", () => {
    expect(matchLocations(paths, "gavetas")).toEqual([3, 4]);
    expect(matchLocations(paths, "drawer 3")).toEqual([3]);
    expect(matchLocations(paths, "workshop")).toEqual([5, 6]);
    expect(matchLocations(paths, "cabinet")).toEqual([2, 3, 4]);
  });
  it("finds a label code", () => {
    expect(matchLocations(paths, "L6")).toEqual([6]);
    expect(matchLocations(paths, "")).toEqual([]);
  });
});
