import { type FormEvent, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ApiError, get } from "../api";
import { Scanner } from "../components/Scanner";
import { ErrorBox, Page } from "../components/ui";

interface Resolved {
  code: string;
  path: string;
}

export function ScanPage() {
  const navigate = useNavigate();
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [typed, setTyped] = useState("");

  async function open(text: string) {
    if (busy) return;
    setBusy(true);
    setMessage(null);
    // The QR payload is a URL ending in the code; Data Matrix holds the code.
    const c = text.trim().split("/").pop() ?? "";
    try {
      const r = await get<Resolved>(`/api/codes/${encodeURIComponent(c)}`);
      navigate(r.path);
    } catch (e) {
      setMessage(e instanceof ApiError ? e.message : `“${text}” is not one of our labels.`);
      setBusy(false);
    }
  }

  function submit(e: FormEvent) {
    e.preventDefault();
    if (typed.trim()) open(typed);
  }

  return (
    <Page title="Scan" subtitle="Point the camera at a label on a drawer, a box or a tool.">
      <div className="stack" style={{ maxWidth: 640, margin: "0 auto" }}>
        <Scanner onCode={open} paused={busy} />
        {message && <ErrorBox error={message} />}
        <div className="divider">or type the code</div>
        <form className="row" style={{ flexWrap: "nowrap" }} onSubmit={submit}>
          <input
            placeholder="e.g. A107 or L42"
            value={typed}
            onChange={(e) => setTyped(e.target.value)}
            autoCapitalize="characters"
            className="mono"
            aria-label="Label code"
          />
          <button type="submit" className="btn btn-primary">
            Open
          </button>
        </form>
      </div>
    </Page>
  );
}
