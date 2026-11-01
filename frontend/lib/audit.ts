export type Outcome = {
  NodeID: string;
  Status: string;
  Note: string;
  Reward: number;
  ScoreDelta: number;
  Deviation: number;
  Weight: number;
};
export type SubLite = {
  value?: number;
  status?: string;
  reward?: number; node_id: string; proof_score: number | null; channel?: string };

export function truncateHash(h: string | undefined | null): string {
  if (!h) return "—";
  return h.length > 18 ? `${h.slice(0, 8)}…${h.slice(-6)}` : h;
}

/** Path for /api/ask with the user's question encoded; null when empty. */
export function askPath(q: string): string | null {
  const t = q.trim();
  return t ? `/api/ask?q=${encodeURIComponent(t)}` : null;
}

export function outcomeRows(outcomes: Outcome[] | null | undefined, subs: SubLite[] | null | undefined) {
  const bySub = new Map<string, SubLite>();
  for (const s of subs ?? []) bySub.set(s.node_id, s);
  return (outcomes ?? []).map((o) => {
    const s = bySub.get(o.NodeID);
    const excluded = o.Status === "excluded";
    const auto = s ? s.proof_score === null : false;
    let reason = o.Note || "";
    if (!reason && auto) reason = "Nœud automatique : source hashée, pas de score de preuve (toujours accepté)";
    else if (!reason) reason = excluded ? "—" : "dans la tolérance";
    return {
      node: o.NodeID,
      excluded,
      badge: excluded ? "EXCLU" : "ACCEPTÉ",
      icon: excluded ? "✗" : "✓",
      reason,
      deviation: `${o.Deviation.toFixed(1)}%`,
      delta: String(o.ScoreDelta),
      reward: o.Reward,
      weight: o.Weight,
      channel: s?.channel || "—",
    };
  });
}
