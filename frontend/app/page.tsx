"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { latestPublication, publicationView, Publication } from "../lib/publication";
import { askPath, outcomeRows, truncateHash, Outcome } from "../lib/audit";
import { ApiConfig, ApiError, apiFetch, latestOnly, resolveApiUrl, singleFlight } from "../lib/api";

type Sub = {
  node_id: string;
  value: number;
  proof_score: number | null;
  channel?: string;
  status: string;
  reward: number;
};

type State = {
  submissions: Sub[];
  publications: Publication[];
  outcomes?: Outcome[];
  rewards: { node_id: string; reward: number; score_delta?: number }[];
};

type Consensus = {
  median: number;
  confidence: number;
  validated: boolean;
  tx: string;
  chain: string;
  value_x100: number;
  aiReportHash: string;
};

export default function Page() {
  const [cfg, setCfg] = useState<ApiConfig | null>(null);
  const [state, setState] = useState<State | null>(null);
  const [loading, setLoading] = useState(true);
  const [answer, setAnswer] = useState("");
  const [question, setQuestion] = useState("prix du mil à Kaolack ?");
  const [copied, setCopied] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState<"" | "consensus" | "ask">("");
  const stateGuard = useRef(latestOnly());
  const askGuard = useRef(latestOnly());

  const refresh = useCallback(async (c: ApiConfig) => {
    const signal = stateGuard.current.next();
    setLoading(true);
    try {
      const r = await apiFetch<State>(c.apiUrl, "/api/state", { signal });
      setState(r.data);
      setError("");
    } catch (e) {
      if (e instanceof DOMException && e.name === "AbortError") return; // superseded
      setError(e instanceof ApiError ? e.message : "Erreur inattendue");
    }
    setLoading(false);
  }, []);

  // Resolve the API URL at runtime (served by /config), then load state.
  const init = useCallback(async () => {
    let c: ApiConfig;
    try {
      const r = await fetch("/config", { cache: "no-store" });
      const j = r.ok ? await r.json() : {};
      c = { apiUrl: resolveApiUrl(j.apiUrl, window.location), token: j.token ?? "" };
    } catch {
      c = { apiUrl: resolveApiUrl(undefined, window.location), token: "" };
    }
    setCfg(c);
    await refresh(c);
  }, [refresh]);

  useEffect(() => {
    init();
  }, [init]);

  const runConsensus = useRef(
    singleFlight(async (c: ApiConfig, set: (b: "" | "consensus" | "ask") => void) => {
      set("consensus");
      setError("");
      setNotice("");
      try {
        const r = await apiFetch<Consensus & { error?: string }>(c.apiUrl, "/api/consensus", {
          method: "POST",
          token: c.token,
          okStatuses: [422],
        });
        if (r.status === 422) {
          setNotice(r.data?.error ?? "Pas assez de sources");
        } else if (!r.data.validated) {
          setNotice(`Consensus non validé (confiance ${r.data.confidence}%) : rien n'a été publié.`);
        }
        await refresh(c);
      } catch (e) {
        setError(e instanceof ApiError ? e.message : "Erreur inattendue");
      } finally {
        set("");
      }
    }),
  ).current;

  async function ask() {
    const path = askPath(question);
    if (!cfg || busy || !path) return;
    const signal = askGuard.current.next();
    setBusy("ask");
    try {
      const r = await apiFetch<{ answer: string }>(cfg.apiUrl, path, { signal });
      setAnswer(r.data.answer);
    } catch (e) {
      if (e instanceof DOMException && e.name === "AbortError") return;
      setError(e instanceof ApiError ? e.message : "Erreur inattendue");
    }
    setBusy("");
  }

  async function copy(h: string) {
    try {
      await navigator.clipboard.writeText(h);
      setCopied(h);
    } catch {
      setError("Copie impossible");
    }
  }
  const hashView = (h?: string) =>
    h ? (
      <>
        <code title={h}>{truncateHash(h)}</code>{" "}
        <button onClick={() => copy(h)}>{copied === h ? "Copié" : "Copier"}</button>
      </>
    ) : (
      "—"
    );
  const rows = outcomeRows(state?.outcomes, state?.submissions);

  const fraud = (s: Sub) => s.proof_score !== null && s.proof_score < 50;

  return (
    <main style={{ fontFamily: "sans-serif", padding: 20, maxWidth: 760 }}>
      <h1>AfriOracle — prix du mil à Kaolack</h1>
      <button disabled={!cfg || busy !== ""} onClick={() => cfg && runConsensus(cfg, setBusy)}>
        {busy === "consensus" ? "Consensus en cours…" : "Lancer le consensus"}
      </button>{" "}
      <button disabled={!cfg || loading} onClick={() => cfg && refresh(cfg)}>
        {loading ? "Chargement…" : "Rafraîchir"}
      </button>
      <form onSubmit={(e) => { e.preventDefault(); ask(); }} style={{ marginTop: 12 }}>
        <label>
          Question à l'assistant :{" "}
          <input value={question} onChange={(e) => setQuestion(e.target.value)} maxLength={500} size={40} />
        </label>{" "}
        <button type="submit" disabled={!cfg || busy !== "" || !question.trim()}>
          {busy === "ask" ? "Question en cours…" : "Interroger l'assistant"}
        </button>
      </form>
      {error && (
        <div role="alert" style={{ background: "#fdecea", border: "1px solid #c62828", color: "#b71c1c", padding: 10, marginTop: 12 }}>
          Erreur : {error}{" "}
          <button onClick={() => (cfg ? refresh(cfg) : init())}>Réessayer</button>
        </div>
      )}
      {notice && <p style={{ color: "#8a5a00" }}>{notice}</p>}

      <h2>Soumissions</h2>
      <table border={1} cellPadding={6} style={{ borderCollapse: "collapse" }}>
        <thead>
          <tr>
            <th>Nœud</th>
            <th>Valeur</th>
            <th>Preuve</th>
            <th>Statut</th>
            <th>Récompense</th>
          </tr>
        </thead>
        <tbody>
          {loading && !state && (
            <tr><td colSpan={5}>Chargement…</td></tr>
          )}
          {state && state.submissions.length === 0 && (
            <tr><td colSpan={5}>Aucune soumission pour l'instant.</td></tr>
          )}
          {(state?.submissions ?? []).map((s, i) => (
            <tr key={`${s.node_id}-${i}`} style={{ color: fraud(s) ? "red" : undefined }}>
              <td>{s.node_id}</td>
              <td>{s.value}</td>
              <td>{s.proof_score === null ? "auto" : s.proof_score}</td>
              <td>{fraud(s) ? "✗ " : ""}{s.status}</td>
              <td>{s.reward}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>Dernière publication</h2>
      {(() => {
        const p = latestPublication(state?.publications);
        if (!p) return <p>{loading && !state ? "Chargement…" : "Aucune publication pour l'instant."}</p>;
        const v = publicationView(p);
        return (
          <div data-testid="publication" data-published={v.published} style={{ border: `2px solid ${v.color}`, background: v.background, padding: 10 }}>
            <p style={{ color: v.color, fontWeight: "bold", margin: 0 }}>{v.title}</p>
            <p>Médiane : {p.median} FCFA/kg</p>
            <p>Confiance : {p.confidence}%</p>
            <p>Tx : {p.tx && v.published ? hashView(p.tx) : v.tx} ({v.chain})</p>
            <p>Date : {v.when}</p>
            {p.aiReportHash && <p>aiReportHash : {hashView(p.aiReportHash)}</p>}
          </div>
        );
      })()}

      <h2>Audit du consensus</h2>
      <table border={1} cellPadding={6} style={{ borderCollapse: "collapse" }}>
        <thead>
          <tr><th>Nœud</th><th>Statut</th><th>Motif</th><th>Écart</th><th>Δ score</th><th>Récompense</th><th>Canal</th><th>Poids</th></tr>
        </thead>
        <tbody>
          {rows.length === 0 && (
            <tr><td colSpan={8}>{loading && !state ? "Chargement…" : "Aucun consensus validé pour l'instant."}</td></tr>
          )}
          {rows.map((r) => (
            <tr key={r.node} data-testid={`outcome-${r.node}`} style={{ background: r.excluded ? "#fdecea" : undefined }}>
              <td>{r.node}</td>
              <td><strong>{r.icon} {r.badge}</strong></td>
              <td>{r.reason}</td>
              <td>{r.deviation}</td>
              <td>{r.delta}</td>
              <td>{r.reward}</td>
              <td>{r.channel}</td>
              <td>{r.weight}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <h2>Récompenses</h2>
      <table border={1} cellPadding={6} style={{ borderCollapse: "collapse" }}>
        <thead>
          <tr><th>Nœud</th><th>Récompense</th><th>Δ score</th></tr>
        </thead>
        <tbody>
          {(state?.rewards ?? []).length === 0 && (
            <tr><td colSpan={3}>{loading && !state ? "Chargement…" : "Aucune récompense pour l'instant."}</td></tr>
          )}
          {(state?.rewards ?? []).map((r) => (
            <tr key={r.node_id}>
              <td>{r.node_id}</td>
              <td>{r.reward}</td>
              <td>{r.score_delta ?? 0}</td>
            </tr>
          ))}
        </tbody>
      </table>

      {answer && (
        <p>
          <strong>Assistant :</strong> {answer}
        </p>
      )}
    </main>
  );
}
