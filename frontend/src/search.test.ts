import { describe, expect, it } from "vitest";
import { rank, stem, tokens, translations } from "./search";

// The same cases as backend/internal/search, so both sides agree.
const names = [
  "Resistência 10k Ω",
  "Condensador 100 nF",
  "Screw M3x8",
  "Chave de fendas PH1",
  "Alicate de corte",
  "Soldering iron",
  "Ventoinha 40 mm",
  "Potência 500 W",
  "Gaveta 3",
  "Válvula de esfera",
  "Rolamento 608",
];
const find = (q: string) => rank(names, q, (n) => [{ text: n, weight: 3 }]);

describe("search", () => {
  it("folds accents, units and plurals", () => {
    expect(tokens("Resistência 10k Ω · Condensador 0,47 µF")).toEqual([
      "resistencia",
      "10k",
      "ohm",
      "condensador",
      "0.47",
      "uf",
    ]);
    expect(stem("condensadores")).toBe("condensador");
    expect(stem("botoes")).toBe("botao");
    expect(find("RESISTÊNCIAS")).toEqual(["Resistência 10k Ω"]);
    expect(find("10kohm")).toEqual(["Resistência 10k Ω"]);
    expect(find("100nf")).toEqual(["Condensador 100 nF"]);
  });
  it("matches Portuguese and English", () => {
    expect(find("resistor")).toEqual(["Resistência 10k Ω"]);
    expect(find("capacitors")).toEqual(["Condensador 100 nF"]);
    expect(find("capac")).toEqual(["Condensador 100 nF"]);
    expect(find("parafusos m3")).toEqual(["Screw M3x8"]);
    expect(find("screwdriver")).toEqual(["Chave de fendas PH1"]);
    expect(find("pliers")).toEqual(["Alicate de corte"]);
    expect(find("solda")).toEqual(["Soldering iron"]);
    expect(find("fan")).toEqual(["Ventoinha 40 mm"]);
    expect(find("drawer")).toEqual(["Gaveta 3"]);
    expect(find("valves")).toEqual(["Válvula de esfera"]);
    expect(find("bearing")).toEqual(["Rolamento 608"]);
  });
  it("needs a short translation to be a whole word", () => {
    // "potenciometro" -> "pot", which must not find "Potência".
    expect(find("potenciometro")).toEqual([]);
  });
  it("lists each translation once, without the word itself", () => {
    const t = translations("solder");
    expect(new Set(t).size).toBe(t.length);
    expect(t).not.toContain("solder");
    expect(t).toContain("solda");
  });
  it("finds nothing for nothing", () => {
    expect(find("")).toEqual([]);
    expect(find("nothing like this")).toEqual([]);
  });
});
