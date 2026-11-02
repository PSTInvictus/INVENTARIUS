import { useQueryClient } from "@tanstack/react-query";
import { Plus, X } from "lucide-react";
import { type FormEvent, useState } from "react";
import { Link } from "react-router-dom";
import { useT } from "../i18n";
import { errorMessage, isUnitKind, useAllItems } from "../lib";
import { addToItem, emptyAdd } from "./AddStock";
import { ItemSelect } from "./pickers";
import { ErrorBox, Sheet, useToast } from "./ui";

interface Row {
  key: number;
  item: number | null;
  amount: string;
  error?: string;
}

let nextKey = 1;
const newRow = (): Row => ({ key: nextKey++, item: null, amount: "" });

/** Puts stock of several items, or new units of tools, in one place at once. */
export function AddHereSheet(props: {
  open: boolean;
  onClose: () => void;
  locationId: number;
  path: string;
}) {
  const [rows, setRows] = useState<Row[]>(() => [newRow()]);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const { data: items } = useAllItems();
  const qc = useQueryClient();
  const toast = useToast();
  const t = useT();

  const filled = rows.filter((r) => r.item != null);
  const update = (key: number, patch: Partial<Row>) =>
    setRows((rs) => {
      const next = rs.map((r) => (r.key === key ? { ...r, ...patch, error: undefined } : r));
      // Picking an item in the last row opens the next one.
      return next[next.length - 1].item != null ? [...next, newRow()] : next;
    });

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    if (filled.length === 0) {
      setError(new Error(t.loc.nothingToAdd));
      return;
    }
    setBusy(true);
    const failed: Row[] = [];
    let done = 0;
    for (const r of filled) {
      const item = items?.find((it) => it.id === r.item);
      if (!item) continue;
      try {
        await addToItem(item, { ...emptyAdd(props.locationId), amount: r.amount });
        done++;
      } catch (err) {
        failed.push({ ...r, error: errorMessage(err) });
      }
    }
    setBusy(false);
    if (done) qc.invalidateQueries();
    if (failed.length === 0) {
      toast(t.loc.addedHere(done));
      props.onClose();
      return;
    }
    setRows([...failed, newRow()]);
    if (done) toast(t.loc.addedHere(done));
    setError(new Error(t.loc.someFailed(failed.length)));
  }

  return (
    <Sheet
      open={props.open}
      onClose={props.onClose}
      title={t.loc.addHere}
      subtitle={t.loc.addHereSubtitle(props.path)}
    >
      <form onSubmit={submit} className="form">
        <div className="add-here-rows">
          {rows.map((r, i) => {
            const item = items?.find((it) => it.id === r.item);
            const units = item ? isUnitKind(item.kind) : false;
            const unit = item && !units && item.unit !== "pcs" ? item.unit : "";
            return (
              <div key={r.key} className="add-here-row">
                <div className="add-here-item">
                  <ItemSelect
                    value={r.item}
                    onChange={(id) => update(r.key, { item: id, amount: r.amount || "1" })}
                    autoFocus={i === 0}
                    aria-label={t.common.item}
                  />
                  {r.error && <span className="field-hint text-danger">{r.error}</span>}
                </div>
                <input
                  className="add-here-amount"
                  inputMode="decimal"
                  aria-label={units ? t.add.howManyUnits : t.loc.amount}
                  placeholder={unit || (units ? "1" : "0")}
                  value={r.amount}
                  disabled={r.item == null}
                  onChange={(e) => update(r.key, { amount: e.target.value })}
                  onKeyDown={(e) => {
                    // Enter goes on to the next item rather than sending the list.
                    if (e.key !== "Enter") return;
                    e.preventDefault();
                    const rowsEl = e.currentTarget.closest(".add-here-rows");
                    const boxes = rowsEl?.querySelectorAll<HTMLInputElement>(".add-here-item input");
                    boxes?.[i + 1]?.focus();
                  }}
                />
                <button
                  type="button"
                  className="btn btn-ghost icon-btn btn-sm"
                  aria-label={t.loc.removeRow}
                  title={t.loc.removeRow}
                  disabled={rows.length === 1 && r.item == null}
                  onClick={() =>
                    setRows((rs) => {
                      const left = rs.filter((x) => x.key !== r.key);
                      return left.length ? left : [newRow()];
                    })
                  }
                >
                  <X size={16} />
                </button>
              </div>
            );
          })}
        </div>
        <div className="row" style={{ justifyContent: "flex-end" }}>
          <Link className="btn btn-sm btn-ghost" to={`/items/new?location=${props.locationId}`}>
            <Plus size={14} /> {t.loc.newItemHere}
          </Link>
        </div>
        <ErrorBox error={error} />
        <button type="submit" className="btn btn-primary btn-lg btn-block" disabled={busy}>
          {busy ? t.common.saving : t.loc.addAll(filled.length)}
        </button>
      </form>
    </Sheet>
  );
}
