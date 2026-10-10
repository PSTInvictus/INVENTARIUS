import { useQueryClient } from "@tanstack/react-query";
import { Printer } from "lucide-react";
import { useState } from "react";
import { api, type Symbology } from "../api";
import { errorMessage, useMeta } from "../lib";
import { useToast } from "./ui";

const DOTS_PER_MM = 180 / 25.4;

const symbologyHelp: Record<Symbology, string> = {
  datamatrix: "Smallest. Scan it with the app.",
  qr: "Any phone camera opens it.",
  code128: "Barcode. Long, for shelves.",
};

/** Shows the label as it will print, with a button to print it. "bare"
 * renders it without its own card (to sit inside another card). */
export function LabelCard({ code, caption, bare }: { code: string; caption?: string; bare?: boolean }) {
  const { data: meta } = useMeta();
  const [symbology, setSymbology] = useState<Symbology | null>(null);
  const [withCaption, setWithCaption] = useState(false);
  const [lengthMM, setLengthMM] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const toast = useToast();
  const qc = useQueryClient();

  const sym = symbology ?? meta?.symbology ?? "datamatrix";
  const cap = withCaption && caption ? caption : "";
  const src = `/api/labels/${code}?symbology=${sym}&caption=${encodeURIComponent(cap)}`;

  async function print() {
    setBusy(true);
    try {
      await api("POST", "/api/print", { labels: [{ code, symbology: sym, caption: cap }] });
      toast(`Label ${code} sent to the printer`);
      qc.invalidateQueries({ queryKey: ["print-jobs"] });
    } catch (e) {
      toast(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className={bare ? "label-card" : "card label-card"}>
      <div className="label-preview">
        <img
          src={src}
          alt={`Label ${code}`}
          onLoad={(e) => setLengthMM(e.currentTarget.naturalWidth / DOTS_PER_MM)}
        />
      </div>
      <div className="label-controls">
        <div className="segmented" role="radiogroup" aria-label="Code type">
          {(["datamatrix", "qr", "code128"] as const).map((s) => (
            <button
              key={s}
              type="button"
              role="radio"
              aria-checked={sym === s}
              className={sym === s ? "on" : ""}
              onClick={() => setSymbology(s)}
            >
              {s === "datamatrix" ? "Data Matrix" : s === "qr" ? "QR" : "Barcode"}
            </button>
          ))}
        </div>
        <div className="muted small">
          {symbologyHelp[sym]} {lengthMM && `Label ≈ ${lengthMM.toFixed(0)} mm.`}
        </div>
        {caption && (
          <label className="check">
            <input type="checkbox" checked={withCaption} onChange={(e) => setWithCaption(e.target.checked)} />
            Add “{caption.length > 24 ? `${caption.slice(0, 24)}…` : caption}”
          </label>
        )}
        <button
          type="button"
          className="btn btn-primary"
          onClick={print}
          disabled={busy}
          style={{ alignSelf: "flex-start" }}
        >
          <Printer size={16} /> Print label {code}
        </button>
      </div>
    </div>
  );
}
