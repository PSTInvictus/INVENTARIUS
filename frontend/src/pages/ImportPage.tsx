import { useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, FileSpreadsheet, Upload } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { api, type ImportResult } from "../api";
import { Card, ErrorBox, Page } from "../components/ui";
import { errorMessage } from "../lib";

const example = `name,kind,category,package,quantity,location,owner,reserved,reserved_note
Resistência 10k Ω,consumable,Eletrónica/Resistências,0603,38,Núcleo PST/Caixa 1/Gaveta 6,,,
Porca M8,consumable,Estruturas/Fixação,,+++,Sala FADEUP/Estruturas,INVICTUS/Structures,,
Parafuso M4x12,consumable,Estruturas/Fixação,,52,Sala FADEUP/Estruturas,INVICTUS/Structures,all,Radax
Multímetro,tool,Ferramentas,,3,Núcleo PST/Parede de ferramentas,,,`;

const columns: [string, string][] = [
  ["name", "Required. What it is."],
  ["kind", "consumable (default), tool, hazmat or flight_hw."],
  ["category", "A path, e.g. Eletrónica/Sensores. Created if missing."],
  ["package", "Footprint, e.g. 0603."],
  ["manufacturer, mpn", "Maker and part number."],
  ["quantity", "12, 9+1, ~10, 5 m, or +++ / muitos for “lots”."],
  ["location", "A path, e.g. Núcleo PST/Caixa 1/Gaveta 5."],
  ["owner", "Empty (team), INVICTUS, or ICARUS/Telemetry."],
  ["status", "For tools: available or broken."],
  ["reserved, reserved_note", "A number or “all”, and what it's for."],
  ["unit, min_stock, lot, expires_on, serial, description, datasheet_url, supplier_url", "Optional extras."],
];

export function ImportPage() {
  const [csv, setCsv] = useState("");
  const [fileName, setFileName] = useState("");
  const [result, setResult] = useState<ImportResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const qc = useQueryClient();

  async function send(commit: boolean) {
    setBusy(true);
    setError(null);
    try {
      const r = await api<ImportResult>("POST", `/api/import/items${commit ? "?commit=1" : ""}`, csv);
      setResult(r);
      if (r.committed) qc.invalidateQueries();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  const bad = result?.rows.filter((r) => r.errors.length > 0) ?? [];
  const lines = csv.trim() ? csv.trim().split("\n").length - 1 : 0;

  return (
    <Page title="Import" subtitle="Bring in a spreadsheet saved as CSV.">
      <div className="layout-2">
        <div className="stack" style={{ gap: "1.25rem" }}>
          <Card title="1 · Choose a file">
            <div className="card-pad stack">
              <label
                className="empty"
                style={{
                  border: "1.5px dashed var(--border-strong)",
                  borderRadius: "var(--r-lg)",
                  cursor: "pointer",
                  background: "var(--surface-2)",
                }}
              >
                <span className="empty-icon" style={{ background: "var(--surface)" }}>
                  <Upload size={20} />
                </span>
                <strong>{fileName || "Choose a CSV file"}</strong>
                <span>
                  {fileName ? `${lines} rows` : "From Excel or Google Sheets: File → Download → CSV"}
                </span>
                <input
                  type="file"
                  accept=".csv,text/csv"
                  hidden
                  onChange={async (e) => {
                    const f = e.target.files?.[0];
                    if (f) {
                      setCsv(await f.text());
                      setFileName(f.name);
                      setResult(null);
                    }
                  }}
                />
              </label>
              <details>
                <summary className="small muted" style={{ cursor: "pointer" }}>
                  …or paste the CSV text
                </summary>
                <textarea
                  className="mono"
                  rows={8}
                  style={{ marginTop: "0.6rem", fontSize: "0.8rem" }}
                  value={csv}
                  onChange={(e) => {
                    setCsv(e.target.value);
                    setFileName("");
                    setResult(null);
                  }}
                  placeholder={example}
                />
                <button
                  type="button"
                  className="btn btn-sm btn-ghost"
                  style={{ marginTop: "0.4rem" }}
                  onClick={() => {
                    setCsv(example);
                    setFileName("");
                    setResult(null);
                  }}
                >
                  Use the example
                </button>
              </details>
            </div>
          </Card>

          <Card title="2 · Check and import">
            <div className="card-pad stack">
              <p className="small muted">
                Checking changes nothing. The import only runs when every row is valid. Importing the same
                file twice adds the quantities twice.
              </p>
              <div className="row">
                <button
                  type="button"
                  className="btn"
                  disabled={!csv.trim() || busy}
                  onClick={() => send(false)}
                >
                  <FileSpreadsheet size={15} /> Check
                </button>
                <button
                  type="button"
                  className="btn btn-primary"
                  disabled={!result?.valid || result.committed || busy}
                  onClick={() => send(true)}
                >
                  <Upload size={15} /> Import{" "}
                  {result?.valid && !result.committed && `${result.rows.length} rows`}
                </button>
              </div>
              <ErrorBox error={error} />
              {result?.committed && (
                <div className="alert alert-ok">
                  <CheckCircle2 size={16} />
                  <span>
                    Imported {result.rows.length} rows ({result.created_items} new items). Print their labels
                    from <Link to="/locations">Locations</Link> or <Link to="/print">Labels</Link>.
                  </span>
                </div>
              )}
              {result && !result.committed && result.valid && (
                <div className="alert alert-ok">
                  <CheckCircle2 size={16} />
                  <span>All {result.rows.length} rows look good.</span>
                </div>
              )}
            </div>
            {bad.length > 0 && (
              <>
                <div className="card-header" style={{ borderTop: "1px solid var(--border)" }}>
                  <h2 className="text-danger">
                    Rows to fix <span className="count">{bad.length}</span>
                  </h2>
                </div>
                <div className="list">
                  {bad.map((r) => (
                    <div key={r.line} className="list-row">
                      <span className="badge mono">line {r.line}</span>
                      <div className="row-main">
                        <div className="row-title">{r.name || "(no name)"}</div>
                        <div className="small text-danger">{r.errors.join(" · ")}</div>
                      </div>
                    </div>
                  ))}
                </div>
              </>
            )}
          </Card>
        </div>

        <Card title="Columns">
          <div className="card-pad stack-sm">
            <p className="small muted">
              The first row names the columns, in any order. Only name is required.
            </p>
            <dl className="facts small" style={{ gridTemplateColumns: "1fr", gap: "0.1rem" }}>
              {columns.map(([c, help]) => (
                <div key={c} style={{ padding: "0.35rem 0", borderBottom: "1px solid var(--border)" }}>
                  <dt>
                    <code style={{ color: "var(--text)" }}>{c}</code>
                  </dt>
                  <dd className="muted">{help}</dd>
                </div>
              ))}
            </dl>
          </div>
        </Card>
      </div>
    </Page>
  );
}
