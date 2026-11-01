"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { ApiConfig, ApiError, apiFetch, latestOnly, resolveApiUrl, singleFlight } from "../lib/api";

type Sub = {
  node_id: string;
  value: number;
  proof_score: number | null;
  status: string;
  reward: number;
};

type State = {
  submissions: Sub[];
  publications: { median: number; confidence: number; tx: string; aiReportHash: string; chain: string }[];
  rewards: { node_id: string; reward: number }[];
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
  const [consensus, setConsensus] = useState<Consensus | null>(null);
  const [answer, setAnswer] = useState("");
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
          setConsensus(null);
          setNotice(`Consensus non validé (confiance ${r.data.confidence}%) : rien n'a été publié.`);
        } else {
          setConsensus(r.data);
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
    if (!cfg || busy) return;
    const signal = askGuard.current.next();
    setBusy("ask");
    try {
      const r = await apiFetch<{ answer: string }>(cfg.apiUrl, `/api/ask?q=${encodeURIComponent("prix du mil à Kaolack ?")}`, { signal });
      setAnswer(r.data.answer);
    } catch (e) {
      if (e instanceof DOMException && e.name === "AbortError") return;
      setError(e instanceof ApiError ? e.message : "Erreur inattendue");
    }
    setBusy("");
  }

  const fraud = (s: Sub) => s.proof_score !== null && s.proof_score < 50;

  return (
    <main style={{ fontFamily: "sans-serif", padding: 20, maxWidth: 760 }}>
      <h1>AfriOracle — prix du mil à Kaolack</h1>
      <button disabled={!cfg || busy !== ""} onClick={() => cfg && runConsensus(cfg, setBusy)}>
        {busy === "consensus" ? "Consensus en cours…" : "Lancer le consensus"}
      </button>{" "}
      <button disabled={!cfg || busy !== ""} onClick={ask}>
        {busy === "ask" ? "Question en cours…" : "Interroger l'assistant"}
      </button>{" "}
      <button disabled={!cfg || loading} onClick={() => cfg && refresh(cfg)}>
        {loading ? "Chargement…" : "Rafraîchir"}
      </button>
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
              <td>{s.status}</td>
              <td>{s.reward}</td>
            </tr>
          ))}
        </tbody>
      </table>

      {consensus && (
        <div>
          <h2>Consensus</h2>
          <p>Médiane : {consensus.median} FCFA/kg</p>
          <p>Confiance : {consensus.confidence}%</p>
          <p>Tx : {consensus.tx} ({consensus.chain})</p>
          <p>aiReportHash : {consensus.aiReportHash}</p>
        </div>
      )}

      {answer && (
        <p>
          <strong>Assistant :</strong> {answer}
        </p>
      )}
    </main>
  );
}
