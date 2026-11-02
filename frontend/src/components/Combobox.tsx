import { ChevronDown } from "lucide-react";
import { type ReactNode, useEffect, useId, useRef, useState } from "react";

export interface ComboEntry {
  /** null is the "nothing chosen" entry. */
  id: number | null;
  title: string;
  meta?: string;
  icon?: ReactNode;
  /** Shown as a heading above the first entry of each group. */
  group?: string;
}

/**
 * A text box that filters a list as you type: arrows move, Enter picks,
 * Escape gives up. Used where a plain <select> would be too long to scroll.
 */
export function Combobox(props: {
  value: number | null;
  /** What the box shows while not typing, e.g. the chosen place's path. */
  display: string;
  placeholder: string;
  /** The entries for what was typed ("" when nothing was). */
  entries: (query: string) => ComboEntry[];
  onChange: (id: number | null) => void;
  noResults: string;
  id?: string;
  required?: boolean;
  autoFocus?: boolean;
  "aria-label"?: string;
}) {
  const [open, setOpen] = useState(false);
  // null while not typing: the box then shows the current choice.
  const [query, setQuery] = useState<string | null>(null);
  const [active, setActive] = useState(0);
  const input = useRef<HTMLInputElement>(null);
  const list = useRef<HTMLDivElement>(null);
  const listId = useId();
  const entries = open ? props.entries(query ?? "") : [];

  // A required box with nothing chosen stops the form from being sent.
  useEffect(() => {
    input.current?.setCustomValidity(props.required && props.value == null ? props.placeholder : "");
  }, [props.required, props.value, props.placeholder]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: keep the highlighted row in view
  useEffect(() => {
    list.current?.querySelector(".combo-item.active")?.scrollIntoView({ block: "nearest" });
  }, [active, open]);

  function show() {
    setOpen(true);
    setQuery(null);
    // Start on the current choice, so Enter keeps it.
    const i = props.entries("").findIndex((e) => e.id === props.value);
    setActive(Math.max(i, 0));
  }

  function close() {
    setOpen(false);
    setQuery(null);
  }

  function pick(e: ComboEntry | undefined) {
    if (!e) return;
    props.onChange(e.id);
    close();
  }

  let lastGroup: string | undefined;
  return (
    <div className={`combo ${open ? "open" : ""}`}>
      <div className="combo-box">
        <input
          ref={input}
          id={props.id}
          role="combobox"
          aria-expanded={open}
          aria-controls={listId}
          aria-autocomplete="list"
          aria-label={props["aria-label"]}
          autoComplete="off"
          autoFocus={props.autoFocus}
          placeholder={props.placeholder}
          value={query ?? props.display}
          onFocus={(e) => {
            show();
            e.target.select();
          }}
          onClick={() => !open && show()}
          onBlur={close}
          onChange={(e) => {
            setOpen(true);
            setQuery(e.target.value);
            setActive(0);
          }}
          onKeyDown={(e) => {
            if (e.key === "ArrowDown") {
              e.preventDefault();
              if (!open) show();
              else setActive((a) => Math.min(a + 1, entries.length - 1));
            } else if (e.key === "ArrowUp") {
              e.preventDefault();
              setActive((a) => Math.max(a - 1, 0));
            } else if (e.key === "Enter" && open) {
              e.preventDefault();
              pick(entries[active]);
            } else if (e.key === "Escape" && open) {
              // Closes the list, not the sheet around it.
              e.preventDefault();
              e.stopPropagation();
              close();
            } else if (e.key === "Tab" && open && query) {
              pick(entries[active]);
            }
          }}
        />
        <ChevronDown size={16} className="combo-chevron" aria-hidden />
      </div>
      {open && (
        // Clicks here must not reach a <label> around the box, which would
        // focus it again; mousedown is cancelled so the box keeps focus.
        // biome-ignore lint/a11y/useKeyWithClickEvents: the keyboard is handled on the input
        <div
          className="combo-list"
          id={listId}
          role="listbox"
          ref={list}
          onMouseDown={(e) => e.preventDefault()}
          onClick={(e) => e.preventDefault()}
        >
          {entries.length === 0 && <div className="combo-empty">{props.noResults}</div>}
          {entries.map((e, i) => {
            const header = e.group !== lastGroup ? e.group : undefined;
            lastGroup = e.group;
            return (
              <div key={`${e.group ?? ""}-${e.id ?? "none"}`}>
                {header && <div className="combo-group">{header}</div>}
                {/* biome-ignore lint/a11y/useKeyWithClickEvents: the keyboard is handled on the input */}
                <div
                  className={`combo-item ${i === active ? "active" : ""} ${e.id === props.value ? "chosen" : ""}`}
                  role="option"
                  aria-selected={e.id === props.value}
                  tabIndex={-1}
                  onMouseMove={() => setActive(i)}
                  onClick={() => pick(e)}
                >
                  {e.icon}
                  <div className="combo-text">
                    <div className="combo-title">{e.title}</div>
                    {e.meta && <div className="combo-meta">{e.meta}</div>}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
