"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { latestPublication, publicationView, Publication } from "../lib/publication";
import { askPath, outcomeRows, truncateHash, Outcome } from "../lib/audit";
import { formatNumber, freshness } from "../lib/format";
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
        <button className="small" onClick={() => copy(h)} aria-label={`Copier ${h}`}>{copied === h ? "Copié" : "Copier"}</button>
      </>
    ) : (
      "—"
    );
  const rows = outcomeRows(state?.outcomes, state?.submissions);

  const fraud = (s: Sub) => s.proof_score !== null && s.proof_score < 50;

  return (
    <>
    <a className="skip-link" href="#main">Aller au contenu</a>
    <main id="main" className="page">
      <h1>AfriOracle — prix du mil à Kaolack</h1>
      <div className="toolbar" role="group" aria-label="Actions">
        <button disabled={!cfg || busy !== ""} onClick={() => cfg && runConsensus(cfg, setBusy)}>
          {busy === "consensus" ? "Consensus en cours…" : "Lancer le consensus"}
        </button>
        <button disabled={!cfg || loading} onClick={() => cfg && refresh(cfg)}>
          {loading ? "Chargement…" : "Rafraîchir"}
        </button>
      </div>
      <form className="ask" onSubmit={(e) => { e.preventDefault(); ask(); }}>
        <label>
          Question à l&apos;assistant :
          <input value={question} onChange={(e) => setQuestion(e.target.value)} maxLength={500} />
        </label>
        <button type="submit" disabled={!cfg || busy !== "" || !question.trim()}>
          {busy === "ask" ? "Question en cours…" : "Interroger l'assistant"}
        </button>
      </form>
      <div role="alert">
        {error && (
          <div className="banner">
            Erreur : {error}{" "}
            <button onClick={() => (cfg ? refresh(cfg) : init())}>Réessayer</button>
          </div>
        )}
      </div>
      <div role="status" aria-live="polite">
        {notice && <p className="notice">{notice}</p>}
        {busy === "consensus" && <span className="sr-only">Consensus en cours</span>}
      </div>

      <h2>Soumissions</h2>
      <div className="table-wrap" tabIndex={0} role="region" aria-label="Soumissions">
      <table>
        <caption className="sr-only">Soumissions des nœuds</caption>
        <thead>
          <tr>
            <th scope="col">Nœud</th>
            <th scope="col">Valeur</th>
            <th scope="col">Preuve</th>
            <th scope="col">Statut</th>
            <th scope="col">Récompense</th>
          </tr>
        </thead>
        <tbody>
          {loading && !state && (
            <tr><td colSpan={5}>Chargement…</td></tr>
          )}
          {state && state.submissions.length === 0 && (
            <tr><td colSpan={5}>Aucune soumission pour l&apos;instant.</td></tr>
          )}
          {(state?.submissions ?? []).map((s, i) => (
            <tr key={`${s.node_id}-${i}`} className={fraud(s) ? "fraud" : undefined}>
              <th scope="row">{s.node_id}</th>
              <td>{formatNumber(s.value)}</td>
              <td>{s.proof_score === null ? "auto" : s.proof_score}</td>
              <td>{fraud(s) ? "✗ " : ""}{s.status}</td>
              <td>{s.reward}</td>
            </tr>
          ))}
        </tbody>
      </table>
      </div>

      <h2>Dernière publication</h2>
      {(() => {
        const p = latestPublication(state?.publications);
        if (!p) return <p>{loading && !state ? "Chargement…" : "Aucune publication pour l'instant."}</p>;
        const v = publicationView(p);
        return (
          <div data-testid="publication" data-published={v.published} className={`publication${v.published ? "" : " unpublished"}`}>
            <p className="title">{v.published ? "✓ " : "✗ "}{v.title}</p>
            <p>Médiane : {formatNumber(p.median)} FCFA/kg</p>
            <p>Confiance : {p.confidence}%</p>
            <p>Tx : {p.tx && v.published ? hashView(p.tx) : v.tx} ({v.chain})</p>
            <p>Date : {v.when}{freshness(p.timestamp) && ` (${freshness(p.timestamp)})`}</p>
            {p.aiReportHash && <p>aiReportHash : {hashView(p.aiReportHash)}</p>}
          </div>
        );
      })()}

      <h2>Audit du consensus</h2>
      <div className="table-wrap" tabIndex={0} role="region" aria-label="Audit du consensus">
      <table>
        <caption className="sr-only">Audit du consensus par nœud</caption>
        <thead>
          <tr><th scope="col">Nœud</th><th scope="col">Statut</th><th scope="col">Motif</th><th scope="col">Écart</th><th scope="col">Δ score</th><th scope="col">Récompense</th><th scope="col">Canal</th><th scope="col">Poids</th></tr>
        </thead>
        <tbody>
          {rows.length === 0 && (
            <tr><td colSpan={8}>{loading && !state ? "Chargement…" : "Aucun consensus validé pour l'instant."}</td></tr>
          )}
          {rows.map((r) => (
            <tr key={r.node} data-testid={`outcome-${r.node}`} className={r.excluded ? "excluded" : undefined}>
              <th scope="row">{r.node}</th>
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
      </div>

      <h2>Récompenses</h2>
      <div className="table-wrap" tabIndex={0} role="region" aria-label="Récompenses">
      <table>
        <caption className="sr-only">Récompenses par nœud</caption>
        <thead>
          <tr><th scope="col">Nœud</th><th scope="col">Récompense</th><th scope="col">Δ score</th></tr>
        </thead>
        <tbody>
          {(state?.rewards ?? []).length === 0 && (
            <tr><td colSpan={3}>{loading && !state ? "Chargement…" : "Aucune récompense pour l'instant."}</td></tr>
          )}
          {(state?.rewards ?? []).map((r) => (
            <tr key={r.node_id}>
              <th scope="row">{r.node_id}</th>
              <td>{r.reward}</td>
              <td>{r.score_delta ?? 0}</td>
            </tr>
          ))}
        </tbody>
      </table>
      </div>

      <div role="status" aria-live="polite">
        {answer && (
          <p className="assistant">
            <strong>Assistant :</strong> {answer}
          </p>
        )}
      </div>
    </main>
    </>
  );
}
