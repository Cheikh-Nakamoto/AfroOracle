"use client";

import { useCallback, useEffect, useState } from "react";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8000";

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
  const [state, setState] = useState<State | null>(null);
  const [consensus, setConsensus] = useState<Consensus | null>(null);
  const [answer, setAnswer] = useState("");
  const [error, setError] = useState("");

  const refresh = useCallback(async () => {
    const r = await fetch(`${API}/api/state`);
    setState(await r.json());
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  async function runConsensus() {
    setError("");
    const r = await fetch(`${API}/api/consensus`, { method: "POST" });
    const data = await r.json();
    if (r.status === 422) {
      setError(data.error ?? "Pas assez de sources");
      return;
    }
    setConsensus(data);
    await refresh();
  }

  async function ask() {
    const r = await fetch(`${API}/api/ask?q=prix du mil à Kaolack ?`);
    const d = await r.json();
    setAnswer(d.answer);
  }

  const fraud = (s: Sub) => s.proof_score !== null && s.proof_score < 50;

  return (
    <main style={{ fontFamily: "sans-serif", padding: 20, maxWidth: 760 }}>
      <h1>AfriOracle — prix du mil à Kaolack</h1>
      <button onClick={runConsensus}>Lancer le consensus</button>{" "}
      <button onClick={ask}>Interroger l'assistant</button>
      {error && <p style={{ color: "red" }}>{error}</p>}

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
          {(state?.submissions ?? []).map((s) => (
            <tr key={s.node_id} style={{ color: fraud(s) ? "red" : undefined }}>
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
