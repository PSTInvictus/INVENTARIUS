import { useQuery } from "@tanstack/react-query";
import {
  AlertTriangle,
  ArrowRight,
  Bookmark,
  Boxes,
  FileUp,
  History as HistoryIcon,
  Plus,
  Printer,
  ScanLine,
  Users,
  Wrench,
} from "lucide-react";
import { Link } from "react-router-dom";
import { get, type MovementRow, type OpenLoan } from "../api";
import { Card, Empty, History, ItemIcon, Stat } from "../components/ui";
import { greeting, isLow, itemTitle, itemTotal, qty, useAllItems, useMe, when } from "../lib";

export function HomePage() {
  const { data: me } = useMe();
  const items = useAllItems();
  const loans = useQuery({ queryKey: ["loans", "all"], queryFn: () => get<OpenLoan[]>("/api/loans") });
  const activity = useQuery({
    queryKey: ["movements", "recent"],
    queryFn: () => get<MovementRow[]>("/api/movements?limit=8"),
  });

  const all = items.data ?? [];
  const low = all.filter(isLow);
  const reserved = all.filter((it) => it.reserved_quantity > 0 || it.reserved_assets > 0).length;
  const mine = loans.data?.filter((l) => l.loan.user_id === me?.id) ?? [];
  const overdue = loans.data?.filter((l) => l.overdue).length ?? 0;

  return (
    <div>
      <div className="welcome">
        <div>
          <h1>{me ? greeting(me.name) : "Welcome"}</h1>
          <div className="page-subtitle">Here's what's happening in the room.</div>
        </div>
      </div>

      {me && me.department_ids.length === 0 && (
        <Link to="/me" className="alert alert-info" style={{ marginBottom: "1rem" }}>
          <Users size={16} />
          <span>
            <strong>Tell us which departments you're in.</strong> It decides what you can edit and fills in
            “for” when you take things. <ArrowRight size={13} style={{ verticalAlign: -2 }} />
          </span>
        </Link>
      )}

      <div className="quick">
        <Link to="/scan" className="primary">
          <span className="row-icon">
            <ScanLine size={18} />
          </span>
          <span>
            Scan a label<small>Take, return, borrow</small>
          </span>
        </Link>
        <Link to="/items/new">
          <span className="row-icon kind-consumable">
            <Plus size={18} />
          </span>
          <span>
            New item<small>Add parts or tools</small>
          </span>
        </Link>
        <Link to="/print">
          <span className="row-icon kind-tool">
            <Printer size={18} />
          </span>
          <span>
            Labels<small>Print and reprint</small>
          </span>
        </Link>
        <Link to="/import">
          <span className="row-icon kind-flight_hw">
            <FileUp size={18} />
          </span>
          <span>
            Import<small>From a spreadsheet</small>
          </span>
        </Link>
      </div>

      <div className="stats" style={{ marginTop: "1.25rem" }}>
        <Stat
          label="Items"
          icon={<Boxes size={14} />}
          value={items.data ? all.length : "–"}
          to="/items"
          hint="in the catalogue"
        />
        <Stat
          label="Running low"
          icon={<AlertTriangle size={14} />}
          value={<span className={low.length ? "text-warn" : ""}>{items.data ? low.length : "–"}</span>}
          to="/items?show=low"
          hint="below their minimum"
        />
        <Stat
          label="On loan"
          icon={<Wrench size={14} />}
          value={loans.data?.length ?? "–"}
          to="/loans"
          hint={overdue ? <span className="text-danger">{overdue} overdue</span> : "tools borrowed"}
        />
        <Stat
          label="Reserved"
          icon={<Bookmark size={14} />}
          value={items.data ? reserved : "–"}
          to="/items?show=reserved"
          hint="items set aside"
        />
      </div>

      <div className="home-grid">
        <Card title={<>Recent activity</>} actions={<HistoryIcon size={16} className="faint" />}>
          {activity.data ? (
            <History rows={activity.data} showItem />
          ) : (
            <div className="card-pad muted">Loading…</div>
          )}
        </Card>

        <div className="stack home-side">
          <Card title="You have" count={mine.length}>
            {mine.length === 0 ? (
              <Empty icon={<Wrench size={20} />}>Nothing borrowed right now.</Empty>
            ) : (
              <div className="list">
                {mine.map((l) => (
                  <Link key={l.loan.id} className="list-row" to={`/assets/${l.loan.asset_id}`}>
                    <ItemIcon kind="tool" />
                    <div className="row-main">
                      <div className="row-title">{l.item_name}</div>
                      <div className="row-meta">
                        <span>A{l.loan.asset_id}</span>
                        <span>since {when(l.loan.taken_at)}</span>
                      </div>
                    </div>
                    {l.overdue && <span className="badge badge-danger">Overdue</span>}
                  </Link>
                ))}
              </div>
            )}
          </Card>

          <Card
            title="Running low"
            count={low.length}
            actions={
              low.length > 0 && (
                <Link to="/items?show=low" className="small">
                  View all
                </Link>
              )
            }
          >
            {low.length === 0 ? (
              <Empty icon={<AlertTriangle size={20} />}>Nothing is below its minimum.</Empty>
            ) : (
              <div className="list">
                {low.slice(0, 6).map((it) => (
                  <Link key={it.id} className="list-row" to={`/items/${it.id}`}>
                    <ItemIcon kind={it.kind} photo={it.photo} />
                    <div className="row-main">
                      <div className="row-title">{itemTitle(it)}</div>
                      <div className="row-meta">
                        <span>min {qty(it.min_stock ?? 0, it.unit)}</span>
                      </div>
                    </div>
                    <div className="row-end">
                      <span className="qty text-warn">{itemTotal(it)}</span>
                    </div>
                  </Link>
                ))}
              </div>
            )}
          </Card>
        </div>
      </div>
    </div>
  );
}
