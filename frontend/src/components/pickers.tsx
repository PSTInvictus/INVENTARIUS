import { useMemo } from "react";
import { useT } from "../i18n";
import {
  fold,
  isUnitKind,
  itemTitle,
  itemTotal,
  kindLabel,
  locationPaths,
  matchLocations,
  recentLocations,
  rememberLocation,
  useAllItems,
  useLocations,
  useMeta,
} from "../lib";
import { rankItems } from "../search";
import { Combobox, type ComboEntry } from "./Combobox";
import { ItemIcon, PlaceIcon } from "./ui";

/** Encodes team / project / department as "", "p:<id>", "d:<id>". */
export function ownerValue(projectId: number | null, departmentId: number | null): string {
  if (departmentId) return `d:${departmentId}`;
  if (projectId) return `p:${projectId}`;
  return "";
}

export function parseOwnerValue(v: string): { project_id: number | null; department_id: number | null } {
  const [k, id] = v.split(":");
  return {
    project_id: k === "p" ? Number(id) : null,
    department_id: k === "d" ? Number(id) : null,
  };
}

/** Select for an owner (team, a project or a department) or a purpose. */
export function OrgSelect(props: {
  value: string;
  onChange: (v: string) => void;
  noneLabel: string;
  id?: string;
}) {
  const { data: meta } = useMeta();
  const t = useT();
  return (
    <select id={props.id} value={props.value} onChange={(e) => props.onChange(e.target.value)}>
      <option value="">{props.noneLabel}</option>
      {meta?.projects.map((p) => (
        <optgroup key={p.id} label={p.name}>
          <option value={`p:${p.id}`}>{t.common.wholeProject(p.name)}</option>
          {meta.departments
            .filter((d) => d.project_id === p.id)
            .map((d) => (
              <option key={d.id} value={`d:${d.id}`}>
                {d.name}
              </option>
            ))}
        </optgroup>
      ))}
    </select>
  );
}

/**
 * Picks a location by typing part of its path ("lab 3", or a label code like
 * "L12"). With nothing typed it lists the places picked last on this device,
 * then every place.
 */
export function LocationSelect(props: {
  value: number | null;
  onChange: (id: number | null) => void;
  emptyLabel?: string;
  exclude?: number;
  id?: string;
  required?: boolean;
  autoFocus?: boolean;
}) {
  const { data: locs } = useLocations();
  const t = useT();
  const paths = useMemo(() => {
    const all = locationPaths(locs);
    if (props.exclude != null) all.delete(props.exclude);
    return all;
  }, [locs, props.exclude]);
  const sorted = useMemo(
    () =>
      [...paths.entries()]
        .sort((a, b) => a[1].localeCompare(b[1], undefined, { numeric: true }))
        .map(([id]) => id),
    [paths],
  );
  const placeholder = props.emptyLabel ?? t.common.chooseLocation;
  // "Not inside anything" and the like, where no location is a valid answer.
  const nullable = props.emptyLabel !== undefined && !props.required;

  const entry = (id: number, group?: string): ComboEntry => {
    const path = paths.get(id) ?? "";
    const cut = path.lastIndexOf(" › ");
    return {
      id,
      group,
      title: cut < 0 ? path : path.slice(cut + 3),
      meta: [cut < 0 ? "" : path.slice(0, cut), `L${id}`].filter(Boolean).join(" · "),
      icon: <PlaceIcon size={15} />,
    };
  };

  function entries(q: string): ComboEntry[] {
    if (q.trim())
      return matchLocations(paths, q)
        .slice(0, 100)
        .map((id) => entry(id));
    const recent = recentLocations().filter((id) => paths.has(id));
    return [
      ...(nullable ? [{ id: null, title: placeholder } as ComboEntry] : []),
      ...recent.map((id) => entry(id, t.common.recentPlaces)),
      ...sorted.map((id) => entry(id, recent.length ? t.common.allPlaces : undefined)),
    ];
  }

  return (
    <Combobox
      id={props.id}
      value={props.value}
      display={props.value != null ? (paths.get(props.value) ?? "") : ""}
      placeholder={placeholder}
      entries={entries}
      required={props.required}
      autoFocus={props.autoFocus}
      noResults={t.common.noPlaceMatches}
      onChange={(id) => {
        if (id != null) rememberLocation(id);
        props.onChange(id);
      }}
    />
  );
}

/** Picks an item by typing part of its name, part number or label code ("I12"). */
export function ItemSelect(props: {
  value: number | null;
  onChange: (id: number | null) => void;
  autoFocus?: boolean;
  "aria-label"?: string;
}) {
  const { data: items } = useAllItems();
  const t = useT();
  const sorted = useMemo(
    () => [...(items ?? [])].sort((a, b) => a.name.localeCompare(b.name, undefined, { numeric: true })),
    [items],
  );
  const current = sorted.find((it) => it.id === props.value);

  function entries(q: string): ComboEntry[] {
    const words = fold(q).split(/\s+/).filter(Boolean);
    const code = words.length === 1 ? /^i(\d+)$/.exec(words[0]) : null;
    const hits = code
      ? sorted.filter((it) => it.id === Number(code[1]))
      : words.length
        ? rankItems(sorted, q)
        : sorted;
    return hits.slice(0, 60).map((it) => ({
      id: it.id,
      title: itemTitle(it),
      meta: [kindLabel(it.kind), it.mpn, `I${it.id}`, isUnitKind(it.kind) ? "" : itemTotal(it)]
        .filter(Boolean)
        .join(" · "),
      icon: <ItemIcon kind={it.kind} photo={it.photo} size={15} />,
    }));
  }

  return (
    <Combobox
      value={props.value}
      display={current ? itemTitle(current) : ""}
      placeholder={t.loc.findItem}
      entries={entries}
      onChange={props.onChange}
      noResults={t.loc.noItemMatches}
      autoFocus={props.autoFocus}
      aria-label={props["aria-label"]}
    />
  );
}
