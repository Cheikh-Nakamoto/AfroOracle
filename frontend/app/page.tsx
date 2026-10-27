"use client";

import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";

const API = "";
const DEMO_QUESTION = "Prix du mil à Kaolack ?";

type Submission = {
  node_id: string;
  tier: number;
  rep_score: number;
  value: number;
  proof_score: number | null;
  channel: string;
  status: string;
  reward: number;
  score_delta: number;
};

type Publication = {
  median: number;
  confidence: number;
  validated: boolean;
  tx: string;
  chain: string;
  value_x100: number;
  aiReportHash: string;
  timestamp: number;
};

type OracleState = {
  submissions: Submission[];
  publications: Publication[];
  rewards: { node_id: string; reward: number; score_delta: number }[];
};

type SubmitResponse = {
  status: string;
  question?: string;
  proof?: { score?: number; model?: string; duplicatePhoto?: boolean };
  submission?: { NodeID: string; Value: number };
};

type AssistantResponse = { answer: string };

async function responseJSON<T>(response: Response): Promise<T> {
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    const details = typeof body.error === "string" ? body.error : "La requête a échoué";
    throw new Error(`${details} (HTTP ${response.status})`);
  }
  return body as T;
}

function formatNumber(value: number, digits = 0) {
  return new Intl.NumberFormat("fr-FR", { maximumFractionDigits: digits }).format(value);
}

function shortHash(value: string) {
  return value.length > 18 ? `${value.slice(0, 10)}…${value.slice(-6)}` : value;
}

function statusLabel(status: string) {
  if (status === "accepted") return "Retenue";
  if (status === "excluded") return "Écartée";
  if (status === "pending") return "En attente";
  return status || "En attente";
}

export default function Page() {
  const [oracle, setOracle] = useState<OracleState | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [working, setWorking] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [assistantQuestion, setAssistantQuestion] = useState(DEMO_QUESTION);
  const [answer, setAnswer] = useState("");
  const [webhookUrl, setWebhookUrl] = useState("https://votre-domaine/webhook/whatsapp");
  const [form, setForm] = useState({ nodeId: "agent-kaolack-01", price: "187", slatePrice: "187", transcript: "" });
  const [photo, setPhoto] = useState<File | null>(null);
  const refreshSequence = useRef(0);

  const refresh = useCallback(async (quiet = false) => {
    const requestSequence = ++refreshSequence.current;
    if (!quiet) setRefreshing(true);
    try {
      const response = await fetch(`${API}/api/state`, { cache: "no-store" });
      const data = await responseJSON<OracleState>(response);
      if (requestSequence !== refreshSequence.current) return;
      if (!Array.isArray(data.submissions) || !Array.isArray(data.publications) || !Array.isArray(data.rewards)) {
        throw new Error("Réponse /api/state invalide.");
      }
      setOracle(data);
      setError("");
    } catch (err) {
      if (requestSequence === refreshSequence.current) setError(err instanceof Error ? err.message : "Impossible de joindre l’API AfriOracle.");
    } finally {
      if (requestSequence === refreshSequence.current) {
        setLoading(false);
        setRefreshing(false);
      }
    }
  }, []);

  useEffect(() => {
    setWebhookUrl(`${window.location.origin}/webhook/whatsapp`);
    void refresh(true);
    const timer = window.setInterval(() => {
      if (document.visibilityState === "visible") void refresh(true);
    }, 5000);
    return () => window.clearInterval(timer);
  }, [refresh]);

  const latest = oracle?.publications.at(-1) ?? null;
  const acceptedCount = useMemo(
    () => (oracle?.submissions ?? []).filter((item) => item.status !== "excluded" && (item.proof_score === null || item.proof_score >= 50)).length,
    [oracle],
  );
  const latestTime = latest ? new Date(latest.timestamp * 1000).toLocaleString("fr-FR", { dateStyle: "medium", timeStyle: "short" }) : "Aucune publication";

  async function submitObservation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setWorking(true);
    setError("");
    setNotice("");
    try {
      const photoB64 = photo ? await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result).split(",")[1] ?? "");
        reader.onerror = () => reject(new Error("La photo n’a pas pu être lue."));
        reader.readAsDataURL(photo);
      }) : "";
      const response = await fetch(`${API}/api/submit`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          node_id: form.nodeId.trim(), tier: 2, rep_score: 65,
          value: Number(form.price), photo_price: form.slatePrice === "" ? undefined : Number(form.slatePrice),
          transcript: form.transcript.trim(), photo_b64: photoB64, channel: "dashboard",
        }),
      });
      const data = await responseJSON<SubmitResponse>(response);
      if (data.status === "clarify") {
        setNotice(data.question ?? "Une précision est nécessaire avant d’enregistrer la soumission.");
      } else {
        const proof = data.proof?.score;
        setNotice(`Observation reçue${proof === undefined ? "" : ` · preuve ${proof}/100`}. Elle apparaîtra dans le consensus après vérification.`);
        setForm((previous) => ({ ...previous, transcript: "" }));
        await refresh(true);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "La soumission n’a pas pu être envoyée.");
    } finally {
      setWorking(false);
    }
  }

  async function runConsensus() {
    setWorking(true);
    setError("");
    setNotice("");
    try {
      const response = await fetch(`${API}/api/consensus`, { method: "POST" });
      const result = await responseJSON<Publication>(response);
      if (typeof result.median !== "number" || typeof result.confidence !== "number" || typeof result.validated !== "boolean") {
        throw new Error("Réponse /api/consensus invalide.");
      }
      await refresh(true);
      setNotice("Le calcul du consensus est terminé. Consultez la publication ci-dessous.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Le consensus n’a pas pu être calculé.");
    } finally {
      setWorking(false);
    }
  }

  async function askOracle(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setWorking(true);
    setError("");
    try {
      const response = await fetch(`${API}/api/ask?q=${encodeURIComponent(assistantQuestion)}`);
      const data = await responseJSON<AssistantResponse>(response);
      if (typeof data.answer !== "string") throw new Error("Réponse /api/ask invalide.");
      setAnswer(data.answer);
    } catch (err) {
      setError(err instanceof Error ? err.message : "L’assistant n’a pas pu répondre.");
    } finally {
      setWorking(false);
    }
  }

  async function copyText(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setNotice("Valeur copiée dans le presse-papiers.");
    } catch {
      setError("Copie impossible dans ce navigateur.");
    }
  }

  return (
    <main className="shell">
      <header className="topbar">
        <a className="brand" href="#top" aria-label="AfriOracle, accueil">
          <span className="brand-mark" aria-hidden="true">A<span>·</span></span>
          <span>AfriOracle<small>Réseau de prix vérifiés</small></span>
        </a>
        <div className="topbar-right">
          <span className="network-dot" /> <span>Kaolack · Marché du mil</span>
          <span className={`api-pill ${error ? "offline" : ""}`}><i />{error ? "API indisponible" : "API connectée"}</span>
        </div>
      </header>

      <section className="hero" id="top">
        <div className="hero-copy">
          <p className="eyebrow">ORACLE AGRICOLE · SÉNÉGAL</p>
          <h1>Le prix juste,<br /><em>porté par le terrain.</em></h1>
          <p className="hero-intro">Des observations locales, vérifiées ensemble. Un prix transparent pour que les décisions d’assurance reposent sur des faits.</p>
          <div className="hero-meta"><span className="live-ring" /> Collecte ouverte <span className="meta-separator">/</span> mise à jour automatique</div>
        </div>
        <div className="price-card">
          <div className="price-card-top"><span>DERNIER PRIX CONSENSUEL</span><span className="grain-icon" aria-hidden="true">✳</span></div>
          {latest ? <>
            <div className="price-value">{formatNumber(latest.median)} <small>FCFA<span>/kg</span></small></div>
            <div className="price-bottom">
              <span className={`validation ${latest.validated ? "" : "unvalidated"}`}><i />{latest.validated ? "Consensus validé" : "Consensus non validé"}</span>
              <span>Confiance <b>{latest.confidence}%</b></span>
            </div>
            <div className="price-time">Publié le {latestTime}</div>
          </> : <>
            <div className="price-empty">— <small>FCFA/kg</small></div>
            <p className="price-time">Aucune valeur publiée pour le moment.</p>
          </>}
        </div>
      </section>

      <section className="metrics" aria-label="État de la collecte">
        <div><span className="metric-label">OBSERVATIONS REÇUES</span><b>{oracle?.submissions.length ?? "—"}</b><small>tous canaux</small></div>
        <div><span className="metric-label">SOURCES ÉLIGIBLES</span><b>{oracle ? acceptedCount : "—"}<small> / 3</small></b><small>minimum pour le consensus</small></div>
        <div><span className="metric-label">PUBLICATIONS</span><b>{oracle?.publications.length ?? "—"}</b><small>{latest?.chain === "simulated" ? "mode de démonstration" : "sur le réseau oracle"}</small></div>
        <button className="refresh-button" onClick={() => void refresh()} disabled={refreshing}>
          <span className={refreshing ? "spin" : ""}>↻</span> {refreshing ? "Actualisation…" : "Actualiser"}
        </button>
      </section>

      {(error || notice) && <div className={`notice ${error ? "notice-error" : "notice-success"}`} role={error ? "alert" : "status"} aria-live={error ? "assertive" : "polite"}>
        <span>{error ? "!" : "✓"}</span><p>{error || notice}</p>{error && <button onClick={() => void refresh()}>Réessayer</button>}
      </div>}

      <div className="workspace-grid">
        <section className="panel observations-panel">
          <div className="section-heading">
            <div><p className="eyebrow">LE TERRAIN</p><h2>Observations récentes</h2></div>
            <span className="count-badge">{oracle?.submissions.length ?? 0} entrées</span>
          </div>
          {loading ? <div className="empty-state"><span className="loading-mark">◌</span><p>Connexion à l’oracle…</p></div> : error && !oracle ? <div className="empty-state"><span className="empty-symbol">!</span><p>Connexion impossible à l’oracle.</p><small>Vérifiez que le serveur API est démarré, puis réessayez.</small></div> : !oracle?.submissions.length ? <div className="empty-state"><span className="empty-symbol">⌁</span><p>Aucune observation pour l’instant.</p><small>Envoyez la première observation ou configurez le webhook WhatsApp.</small></div> :
            <div className="table-wrap"><table>
              <caption className="sr-only">Observations de prix soumises à l’oracle de Kaolack</caption>
              <thead><tr><th scope="col">SOURCE</th><th scope="col">PRIX</th><th scope="col">PREUVE</th><th scope="col">CANAL</th><th scope="col">ÉTAT</th><th scope="col">RÉCOMPENSE</th></tr></thead>
              <tbody>{[...oracle.submissions].reverse().map((item, index) => {
                const excluded = item.status === "excluded" || (item.proof_score !== null && item.proof_score < 50);
                return <tr key={`${item.node_id}-${index}`}>
                  <td><span className="source-name"><span className={`source-avatar ${excluded ? "avatar-alert" : ""}`}>{item.node_id.slice(0, 1).toUpperCase()}</span><span>{item.node_id}<small>Tier {item.tier} · rép. {item.rep_score}</small></span></span></td>
                  <td className="price-cell">{formatNumber(item.value)} <small>FCFA</small></td>
                  <td>{item.proof_score === null ? <span className="proof-auto">Source automatique</span> : <span className={`proof-score ${item.proof_score < 50 ? "proof-low" : ""}`}>{item.proof_score}<small>/100</small></span>}</td>
                  <td><span className="channel-label">{item.channel === "whatsapp" ? "WhatsApp" : item.channel || "—"}</span></td>
                  <td><span className={`status-badge ${excluded ? "status-excluded" : item.status === "accepted" ? "status-accepted" : "status-pending"}`}><i />{excluded && item.proof_score !== null ? `Écartée · preuve ${item.proof_score}` : statusLabel(item.status)}</span>{item.score_delta !== 0 && <small className="score-delta">Réputation {item.score_delta > 0 ? "+" : ""}{formatNumber(item.score_delta, 1)}</small>}</td>
                  <td className="reward-cell">{formatNumber(item.reward, 1)} <small>AFRO</small></td>
                </tr>;
              })}</tbody>
            </table></div>}
          {latest && <div className="publication-strip">
            <div><span className="strip-kicker">DERNIÈRE PUBLICATION</span><strong>{formatNumber(latest.median)} FCFA/kg</strong></div>
            <div className="hash-line"><span>{latest.chain === "simulated" ? "Transaction simulée" : "Transaction Polygon"}</span><code title={latest.tx}>{shortHash(latest.tx)}</code><button aria-label="Copier la transaction" onClick={() => void copyText(latest.tx)}>Copier</button></div>
          </div>}
        </section>

        <aside className="side-column">
          <section className="panel submit-panel">
            <div className="section-heading compact-heading"><div><p className="eyebrow">CONTRIBUTEUR</p><h2>Ajouter un prix</h2></div><span className="panel-icon">＋</span></div>
            <p className="panel-description">Saisissez le prix observé et celui lu sur l’ardoise. L’API attribue un score de preuve.</p>
            <form onSubmit={submitObservation} className="observation-form">
              <label>Identifiant du nœud<input required value={form.nodeId} onChange={(event) => setForm({ ...form, nodeId: event.target.value })} placeholder="agent-kaolack-01" /></label>
              <div className="form-row">
                <label>Prix observé<input required min="1" step="1" type="number" value={form.price} onChange={(event) => setForm({ ...form, price: event.target.value })} /><small>FCFA/kg</small></label>
              <label>Prix sur l’ardoise<input required min="1" step="1" type="number" value={form.slatePrice} onChange={(event) => setForm({ ...form, slatePrice: event.target.value })} /><small>lu sur la photo</small></label>
              </div>
              <label>Photo de preuve<input required type="file" accept="image/*" capture="environment" onChange={(event) => setPhoto(event.target.files?.[0] ?? null)} /><small>{photo ? photo.name : "Photographier l’ardoise du marché"}</small></label>
              <label>Note ou transcription <span className="optional">FACULTATIF</span><textarea rows={2} value={form.transcript} onChange={(event) => setForm({ ...form, transcript: event.target.value })} placeholder="Ex. Le kilo de mil est à 187 francs…" /></label>
              <button className="primary-button" disabled={working}>{working ? "Envoi en cours…" : "Envoyer l’observation"}<span aria-hidden="true">↗</span></button>
            </form>
            <div className="form-footnote"><span>i</span> Démo interface · Tier terrain · réputation 65</div>
          </section>

          <section className="whatsapp-card">
            <div className="whatsapp-head"><span className="wa-icon">◉</span><span><b>Canal WhatsApp</b><small>Réception des observations</small></span><span className="setup-tag">À CONFIGURER</span></div>
            <p>Le backend reçoit les messages texte via un webhook Meta. L’association d’un compte utilisateur et les médias entrants ne sont pas encore implémentés.</p>
            <div className="webhook-row"><code>{webhookUrl.replace(/^https?:\/\//, "")}</code><button onClick={() => void copyText(webhookUrl)}>Copier l’URL</button></div>
            <small className="wa-note">URL publique requise · token de vérification configuré côté serveur</small>
          </section>
        </aside>
      </div>

      <div className="lower-grid">
        <section className="panel consensus-panel">
          <div className="section-heading compact-heading"><div><p className="eyebrow">DÉCISION COLLECTIVE</p><h2>Consensus pondéré</h2></div><span className="consensus-symbol">⌁</span></div>
          <p className="panel-description">Le calcul exige au moins trois sources indépendantes éligibles. Les preuves faibles sont exclues avant le calcul.</p>
          <button className="consensus-button" onClick={() => void runConsensus()} disabled={working || acceptedCount < 3}>{working ? "Traitement…" : "Lancer le consensus"}<span>→</span></button>
          <small className="button-hint">{acceptedCount < 3 ? `Encore ${3 - acceptedCount} source${3 - acceptedCount > 1 ? "s" : ""} éligible${3 - acceptedCount > 1 ? "s" : ""} nécessaire${3 - acceptedCount > 1 ? "s" : ""}` : "Sources suffisantes pour calculer"}</small>
        </section>
        <section className="panel assistant-panel">
          <div className="section-heading compact-heading"><div><p className="eyebrow">ACCÈS AUX DONNÉES</p><h2>Demander à l’oracle</h2></div><span className="assistant-icon">✳</span></div>
          <form className="ask-form" onSubmit={askOracle}><label className="sr-only" htmlFor="ask-question">Question à l’oracle</label><input id="ask-question" value={assistantQuestion} onChange={(event) => setAssistantQuestion(event.target.value)} /><button disabled={working} aria-label="Envoyer la question">↗</button></form>
          <div className={`answer-box ${answer ? "answer-filled" : ""}`} aria-live="polite"><span className="answer-mark">“</span><p>{answer || "La réponse s’appuiera uniquement sur les données publiées par l’oracle."}</p></div>
        </section>
      </div>

      {latest && <section className="integrity-row" aria-label="Preuves de publication">
        <div><span className="integrity-icon">◇</span><span><small>Empreinte du rapport IA</small><code title={latest.aiReportHash}>{shortHash(latest.aiReportHash)}</code></span><button onClick={() => void copyText(latest.aiReportHash)}>Copier</button></div>
        <div><span className="integrity-icon">⌘</span><span><small>Réseau</small><b>{latest.chain === "simulated" ? "Simulation locale" : "Polygon Amoy"}</b></span></div>
        <p>La publication conserve une empreinte vérifiable du rapport. Les médias restent hors chaîne.</p>
      </section>}

      <footer className="footer"><span>AFRIORACLE <i>·</i> LE TERRAIN FAIT FOI</span><span>Prototype hackathon <i>·</i> Kaolack, Sénégal</span></footer>
    </main>
  );
}
