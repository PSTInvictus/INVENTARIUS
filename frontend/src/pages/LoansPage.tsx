import { useQuery } from "@tanstack/react-query";
import { Wrench } from "lucide-react";
import { Link, useNavigate } from "react-router-dom";
import { get, type OpenLoan } from "../api";
import { Card, Empty, ErrorBox, ItemIcon, Loading, Page } from "../components/ui";
import { forLabel, useMe, useMeta, when } from "../lib";

/** The "who has what" board for shared tools. */
export function LoansPage() {
  const loans = useQuery({
    queryKey: ["loans", "all"],
    queryFn: () => get<OpenLoan[]>("/api/loans"),
    refetchInterval: 30_000,
  });
  const { data: meta } = useMeta();
  const { data: me } = useMe();
  const navigate = useNavigate();
  const all = loans.data ?? [];
  const overdue = all.filter((l) => l.overdue);
  const rest = all.filter((l) => !l.overdue);

  const table = (rows: OpenLoan[]) => (
    <>
      <div className="table-wrap only-desktop">
        <table className="table">
          <thead>
            <tr>
              <th>Tool</th>
              <th>Who</th>
              <th>For</th>
              <th>Since</th>
              <th>Due</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((l) => (
              <tr
                key={l.loan.id}
                className="clickable"
                onClick={() => navigate(`/assets/${l.loan.asset_id}`)}
              >
                <td>
                  <div className="cell-main">
                    <ItemIcon kind="tool" size={16} />
                    <div>
                      <Link to={`/assets/${l.loan.asset_id}`} onClick={(e) => e.stopPropagation()}>
                        {l.item_name}
                      </Link>
                      <div className="row-meta">
                        <span className="mono">A{l.loan.asset_id}</span>
                        {l.asset_serial && <span>{l.asset_serial}</span>}
                      </div>
                    </div>
                  </div>
                </td>
                <td>
                  {l.user_name}
                  {l.loan.user_id === me?.id && (
                    <span className="badge badge-info" style={{ marginLeft: 6 }}>
                      You
                    </span>
                  )}
                </td>
                <td className="muted small">
                  {forLabel(meta, l.loan.project_id, l.loan.department_id) || "—"}
                </td>
                <td className="muted small nowrap">{when(l.loan.taken_at)}</td>
                <td className="nowrap">
                  {l.loan.due_at ? (
                    <span className={`badge ${l.overdue ? "badge-danger" : ""}`}>{when(l.loan.due_at)}</span>
                  ) : (
                    <span className="faint">—</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <div className="list only-mobile">
        {rows.map((l) => (
          <Link key={l.loan.id} to={`/assets/${l.loan.asset_id}`} className="list-row">
            <ItemIcon kind="tool" />
            <div className="row-main">
              <div className="row-title">{l.item_name}</div>
              <div className="row-meta">
                <span>{l.loan.user_id === me?.id ? "You" : l.user_name}</span>
                {(l.loan.project_id || l.loan.department_id) && (
                  <span>{forLabel(meta, l.loan.project_id, l.loan.department_id)}</span>
                )}
                <span>{when(l.loan.taken_at)}</span>
              </div>
            </div>
            {l.overdue && <span className="badge badge-danger">Overdue</span>}
          </Link>
        ))}
      </div>
    </>
  );

  return (
    <Page
      title="Loans"
      subtitle={
        loans.data && (all.length ? `${all.length} tools borrowed right now` : "Every tool is in its place")
      }
    >
      <ErrorBox error={loans.error} />
      {loans.isPending ? (
        <Loading />
      ) : all.length === 0 ? (
        <Card>
          <Empty icon={<Wrench size={20} />} title="Nothing on loan">
            When someone borrows a tool, it shows up here.
          </Empty>
        </Card>
      ) : (
        <div className="stack" style={{ gap: "1.25rem" }}>
          {overdue.length > 0 && (
            <Card title={<span className="text-danger">Overdue</span>} count={overdue.length}>
              {table(overdue)}
            </Card>
          )}
          {rest.length > 0 && (
            <Card title="Borrowed" count={rest.length}>
              {table(rest)}
            </Card>
          )}
        </div>
      )}
    </Page>
  );
}
