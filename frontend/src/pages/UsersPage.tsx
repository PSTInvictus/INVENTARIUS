import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Pencil, ShieldCheck } from "lucide-react";
import { useState } from "react";
import { api, get, type Role, type User } from "../api";
import { Card, Empty, ErrorBox, Loading, Page, Sheet, useToast } from "../components/ui";
import { errorMessage, useMe, useMeta } from "../lib";

const roleLabel: Record<Role, string> = { member: "Member", dept_lead: "Department lead", admin: "Admin" };

export function UsersPage() {
  const users = useQuery({ queryKey: ["users"], queryFn: () => get<User[]>("/api/users") });
  const { data: me } = useMe();
  const { data: meta } = useMeta();
  const [editing, setEditing] = useState<User | null>(null);
  const qc = useQueryClient();
  const toast = useToast();

  async function update(u: User, patch: Partial<Pick<User, "role" | "active" | "department_ids">>) {
    try {
      await api("PATCH", `/api/users/${u.id}`, patch);
      await qc.invalidateQueries({ queryKey: ["users"] });
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  }

  if (me?.role !== "admin")
    return (
      <Page title="People">
        <Card>
          <Empty icon={<ShieldCheck size={20} />} title="Admins only">
            Ask an admin to change roles or departments.
          </Empty>
        </Card>
      </Page>
    );
  if (users.isPending) return <Loading />;

  const deptNames = (ids: number[]) =>
    ids
      .map((id) => {
        const d = meta?.departments.find((x) => x.id === id);
        const p = d && meta?.projects.find((x) => x.id === d.project_id);
        return d && p ? `${p.code.slice(0, 3)} · ${d.name}` : "";
      })
      .filter(Boolean);
  const current = editing && users.data?.find((u) => u.id === editing.id);

  return (
    <Page title="People" subtitle="Anyone with a team Google account can sign in, and starts as a member.">
      <ErrorBox error={users.error} />
      <Card count={users.data?.length} title="Team members">
        <div className="list">
          {users.data?.map((u) => (
            <div key={u.id} className="list-row" style={{ opacity: u.active ? 1 : 0.55, flexWrap: "wrap" }}>
              <span className="avatar">
                {u.picture ? (
                  <img src={u.picture} alt="" referrerPolicy="no-referrer" />
                ) : (
                  u.name.slice(0, 1).toUpperCase()
                )}
              </span>
              <div className="row-main" style={{ minWidth: 200 }}>
                <div className="row-title">
                  {u.name}
                  {u.id === me.id && (
                    <span className="badge badge-info" style={{ marginLeft: 6 }}>
                      You
                    </span>
                  )}
                  {!u.active && (
                    <span className="badge" style={{ marginLeft: 6 }}>
                      Disabled
                    </span>
                  )}
                </div>
                <div className="row-meta">
                  <span>{u.email}</span>
                </div>
                <div className="tags" style={{ marginTop: "0.35rem" }}>
                  {deptNames(u.department_ids).map((n) => (
                    <span key={n} className="badge">
                      {n}
                    </span>
                  ))}
                  <button
                    type="button"
                    className="btn btn-ghost btn-sm"
                    style={{ height: 24 }}
                    onClick={() => setEditing(u)}
                  >
                    <Pencil size={12} /> Departments
                  </button>
                </div>
              </div>
              <div className="row" style={{ flexWrap: "nowrap" }}>
                <select
                  value={u.role}
                  onChange={(e) => update(u, { role: e.target.value as Role })}
                  disabled={u.id === me.id}
                  aria-label={`Role of ${u.name}`}
                  style={{ width: "auto" }}
                >
                  {(Object.keys(roleLabel) as Role[]).map((r) => (
                    <option key={r} value={r}>
                      {roleLabel[r]}
                    </option>
                  ))}
                </select>
                <label className="check small" title="Can sign in">
                  <input
                    type="checkbox"
                    checked={u.active}
                    disabled={u.id === me.id}
                    onChange={(e) => update(u, { active: e.target.checked })}
                  />
                  Active
                </label>
              </div>
            </div>
          ))}
        </div>
        <div className="card-footer muted">
          Members take, return and borrow. Department leads will approve purchases. Admins manage people,
          places and the printer.
        </div>
      </Card>

      {current && (
        <Sheet open onClose={() => setEditing(null)} title="Departments" subtitle={current.name}>
          <div className="stack">
            {meta?.projects.map((p) => (
              <div key={p.id} className="stack-sm">
                <h3>{p.name}</h3>
                <div className="chips">
                  {meta.departments
                    .filter((d) => d.project_id === p.id)
                    .map((d) => {
                      const on = current.department_ids.includes(d.id);
                      return (
                        <button
                          key={d.id}
                          type="button"
                          className={`chip ${on ? "on" : ""}`}
                          onClick={() =>
                            update(current, {
                              department_ids: on
                                ? current.department_ids.filter((x) => x !== d.id)
                                : [...current.department_ids, d.id],
                            })
                          }
                        >
                          {d.name}
                        </button>
                      );
                    })}
                </div>
              </div>
            ))}
            <button type="button" className="btn btn-primary btn-block" onClick={() => setEditing(null)}>
              Done
            </button>
          </div>
        </Sheet>
      )}
    </Page>
  );
}
