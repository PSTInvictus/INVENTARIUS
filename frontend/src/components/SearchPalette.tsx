import { useQuery } from "@tanstack/react-query";
import { CornerDownLeft, FileUp, Plus, Printer, QrCode, ScanLine, Search } from "lucide-react";
import {
  createContext,
  type ReactNode,
  useCallback,
  useContext,
  useDeferredValue,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useNavigate } from "react-router-dom";
import { get, type SearchResult } from "../api";
import { itemTitle, itemTotal, kindLabel, locationPaths, useLocations } from "../lib";
import { ItemIcon, PlaceIcon } from "./ui";

interface Entry {
  key: string;
  group: string;
  to: string;
  icon: ReactNode;
  title: string;
  meta?: string;
  end?: string;
}

const PaletteCtx = createContext<() => void>(() => {});
export const useOpenSearch = () => useContext(PaletteCtx);

/** Provides the search palette, opened with "/" or Ctrl/Cmd+K anywhere. */
export function SearchPaletteProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState(false);
  const show = useCallback(() => setOpen(true), []);
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const typing =
        e.target instanceof HTMLElement &&
        (e.target.isContentEditable || /^(INPUT|TEXTAREA|SELECT)$/.test(e.target.tagName));
      if ((e.key === "k" && (e.metaKey || e.ctrlKey)) || (e.key === "/" && !typing)) {
        e.preventDefault();
        setOpen(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
  return (
    <PaletteCtx.Provider value={show}>
      {children}
      {open && <Palette onClose={() => setOpen(false)} />}
    </PaletteCtx.Provider>
  );
}

const shortcuts: Entry[] = [
  { key: "scan", group: "Actions", to: "/scan", icon: <ScanLine size={17} />, title: "Scan a label" },
  { key: "new", group: "Actions", to: "/items/new", icon: <Plus size={17} />, title: "New item" },
  { key: "print", group: "Actions", to: "/print", icon: <Printer size={17} />, title: "Labels & printer" },
  { key: "import", group: "Actions", to: "/import", icon: <FileUp size={17} />, title: "Import from CSV" },
];

function Palette({ onClose }: { onClose: () => void }) {
  const [q, setQ] = useState("");
  const query = useDeferredValue(q.trim());
  const [active, setActive] = useState(0);
  const navigate = useNavigate();
  const listRef = useRef<HTMLDivElement>(null);
  const { data: locs } = useLocations();
  const paths = useMemo(() => locationPaths(locs), [locs]);
  const search = useQuery({
    queryKey: ["search", query],
    queryFn: () => get<SearchResult>(`/api/search?q=${encodeURIComponent(query)}`),
    enabled: query.length > 0,
    placeholderData: (prev) => prev,
  });

  const entries: Entry[] = useMemo(() => {
    if (!query) return shortcuts;
    const r = search.data;
    if (!r) return [];
    const out: Entry[] = [];
    if (r.code)
      out.push({
        key: `code-${r.code.code}`,
        group: "Label",
        to: r.code.path,
        icon: (
          <span className="row-icon kind-consumable">
            <QrCode size={17} />
          </span>
        ),
        title: `Open label ${r.code.code}`,
      });
    for (const it of r.items)
      out.push({
        key: `i${it.id}`,
        group: "Items",
        to: `/items/${it.id}`,
        icon: <ItemIcon kind={it.kind} photo={it.photo} size={16} />,
        title: itemTitle(it),
        meta: [kindLabel[it.kind], it.mpn].filter(Boolean).join(" · "),
        end:
          it.kind === "tool" || it.kind === "flight_hw"
            ? `${it.available_count}/${it.asset_count} free`
            : itemTotal(it),
      });
    for (const l of r.locations)
      out.push({
        key: `l${l.id}`,
        group: "Places",
        to: `/locations/${l.id}`,
        icon: <PlaceIcon size={16} />,
        title: l.name,
        meta: paths.get(l.id),
      });
    return out;
  }, [query, search.data, paths]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: new results reset the selection
  useEffect(() => setActive(0), [entries]);
  // biome-ignore lint/correctness/useExhaustiveDependencies: keep the selected row in view
  useEffect(() => {
    listRef.current?.querySelector(".palette-item.active")?.scrollIntoView({ block: "nearest" });
  }, [active]);

  function go(e: Entry | undefined) {
    if (!e) return;
    onClose();
    navigate(e.to);
  }

  let lastGroup = "";
  return (
    // Clicking the backdrop closes; Escape is handled on the input.
    // biome-ignore lint/a11y/useKeyWithClickEvents: see above
    // biome-ignore lint/a11y/noStaticElementInteractions: see above
    <div className="palette" onClick={(e) => e.target === e.currentTarget && onClose()}>
      <div className="palette-box" role="dialog" aria-label="Search">
        <div className="palette-input">
          <Search size={18} />
          <input
            autoFocus
            placeholder="Search items, part numbers, places, or type a code like A107…"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Escape") onClose();
              else if (e.key === "ArrowDown") {
                e.preventDefault();
                setActive((a) => Math.min(a + 1, entries.length - 1));
              } else if (e.key === "ArrowUp") {
                e.preventDefault();
                setActive((a) => Math.max(a - 1, 0));
              } else if (e.key === "Enter") go(entries[active]);
            }}
          />
          <button type="button" className="btn btn-ghost btn-sm only-mobile" onClick={onClose}>
            Cancel
          </button>
        </div>
        <div className="palette-results" ref={listRef}>
          {query && search.data && entries.length === 0 && (
            <div className="empty">
              <strong>No results for “{query}”</strong>
              <span>Try a part number, a value like “10k 0603”, or a place.</span>
            </div>
          )}
          {entries.map((e, i) => {
            const header = e.group !== lastGroup ? e.group : null;
            lastGroup = e.group;
            return (
              <div key={e.key}>
                {header && <div className="palette-group">{header}</div>}
                {/* biome-ignore lint/a11y/useKeyWithClickEvents: keyboard handled by the input */}
                <div
                  className={`palette-item ${i === active ? "active" : ""}`}
                  role="option"
                  aria-selected={i === active}
                  tabIndex={-1}
                  onMouseMove={() => setActive(i)}
                  onClick={() => go(e)}
                >
                  {e.icon}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div className="row-title">{e.title}</div>
                    {e.meta && <div className="row-meta">{e.meta}</div>}
                  </div>
                  {e.end && <span className="small muted num">{e.end}</span>}
                  {i === active && <CornerDownLeft size={14} className="faint" />}
                </div>
              </div>
            );
          })}
        </div>
        <div className="palette-foot">
          <span>
            <kbd>↑</kbd> <kbd>↓</kbd> to move
          </span>
          <span>
            <kbd>Enter</kbd> to open
          </span>
          <span>
            <kbd>Esc</kbd> to close
          </span>
        </div>
      </div>
    </div>
  );
}
