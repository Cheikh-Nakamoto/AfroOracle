# UI_AUDIT — AfriOracle frontend (Next.js dashboard)

- Scope (read-only re-verified): `frontend/app/page.tsx`, `frontend/app/layout.tsx`,
  `frontend/package.json`, `frontend/tsconfig.json`, `frontend/Dockerfile`, `docker-compose.yml`,
  `scripts/demo.sh`; backend contracts consumed: `backend/internal/controller/{state,consensus,ask,submit}.go`,
  `backend/internal/model/model.go`, `backend/internal/model/consensus.go` (validated rule).
- Verdict: **P0=4, P1=8, P2=6, total=18 (UI-01..UI-18)**. Counts unchanged after re-check.
- Method: line-level grep/read only. No code, config, test, or state files modified by this audit.

## Executive summary

What exists today: a single client page (`frontend/app/page.tsx`, `"use client"` at :1) that
fetches `GET /api/state` once on mount, renders submissions in a 5-column table, runs
`POST /api/consensus` on click, keeps the consensus result only in local React state, and calls
`GET /api/ask` with a hardcoded question. Styling is 100% inline; no components, no CSS file,
no extra routes (`frontend/app/` contains only `page.tsx` + `layout.tsx`).

Top demo risks: (1) any network/HTTP failure leaves a silent empty page with dead-looking
buttons (UI-01); (2) the published result vanishes on reload because `publications`/`rewards`
are fetched and typed but never rendered (UI-02); (3) the API URL defaults to the visitor's
own `localhost`, baked in at build time (UI-03); (4) an unvalidated consensus (`validated:false`)
is displayed exactly like a published one (UI-04); (5) WhatsApp submissions never appear
without a manual page reload (UI-05). The happy path (median 187, D red/excluded) works only
on localhost, with the backend already up, and without touching F5 at the wrong moment.

## Inventory (file:line evidence)

| Area | Evidence | Fact |
|---|---|---|
| API base | `frontend/app/page.tsx:5` | `process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8000"`, browser-side |
| Types | `page.tsx:7-29` | `Sub` omits tier/rep/channel/score_delta; `State.publications` lacks validated/timestamp; `Consensus` omits outcomes |
| State fetch | `page.tsx:37-44` | `refresh()` fetch + `r.json()`, no `r.ok`, no try/catch; `useEffect` runs it once on mount |
| Consensus | `page.tsx:46-56` | POST; only 422 handled (:50-53); any other status falls into `setConsensus(data)` (:54) |
| Ask | `page.tsx:58-62` | hardcoded unencoded `?q=prix du mil à Kaolack ?` (:59); backend ignores `q` (`ask.go:11-22`) |
| Fraud flag | `page.tsx:64,86` | `proof_score < 50` recolor the whole row red; no text badge |
| Render | `page.tsx:66-113` | table :74-95; local-only consensus block :97-105; answer :107-111; publications/rewards never in JSX |
| Layout | `frontend/app/layout.tsx:3-12` | fr lang + metadata OK; only `body{margin:0}` (:11); no global styles |
| Tooling | `frontend/package.json:5-20` | scripts dev/build/start only; no test/lint/typecheck; next 14.2.5, react ^18, TS ^5 |
| Build-time URL | `frontend/Dockerfile:11-13`, `docker-compose.yml:21-27` | `NEXT_PUBLIC_API_URL` ARG/ENV inlined by `next build`; compose default `http://localhost:8000` |
| Backend state | `backend/internal/controller/state.go:19-24` | returns submissions, publications, rewards, fingerprints |
| Backend consensus | `controller/consensus.go:36-41,58-69` | 422 if <3 accepted; stores publication each run; returns validated + outcomes |
| Validated rule | `backend/internal/model/consensus.go:95` | `validated = confidence >= 66` |
| Backend ask | `controller/ask.go:13-22` | no publications => never-estimates message; else quotes last publication |
| Wire types | `backend/internal/model/model.go:37-64` | StateSub, Publication (validated, timestamp), RewardEntry (score_delta) |

## Findings

### UI-01 — P0 — Silent network/HTTP failure: empty page, dead buttons, unhandled rejection
- Evidence: `page.tsx:37-44` (no `r.ok`/try/catch), `:58-62` (same in `ask`); grep: no `AbortController`, no catch.
- Impact: backend down or wrong URL => blank table, buttons appear dead, unhandled promise rejection; demo dies silently.
- Fix: shared fetch helper: check `r.ok`, try/catch, set a visible error state for load/consensus/ask failures.
- Acceptance: stop backend, reload: explicit error banner + retry button; no unhandled rejection in console.

### UI-02 — P0 — Published result disappears on reload; publications/rewards never rendered
- Evidence: typed at `page.tsx:15-19` and returned by `state.go:19-24`, but JSX renders only local `consensus` (:97-105).
- Impact: F5 after consensus erases median/tx from screen though backend kept the publication; rewards invisible too.
- Fix: render latest `state.publications[]` (median, confidence, validated, tx, chain, timestamp) + rewards; seed from refresh.
- Acceptance: run consensus, press F5: same median/tx still displayed from `/api/state`, without re-clicking.

### UI-03 — P0 — `NEXT_PUBLIC_API_URL` localhost baked at build breaks any non-localhost demo
- Evidence: `page.tsx:5`; `frontend/Dockerfile:11-13` (ENV before `next build`); `docker-compose.yml:25-26` default localhost.
- Impact: on LAN IP/domain/projector machine, browser calls the visitor's own `localhost:8000`; UI-01 silence hides why.
- Fix: fail loudly if unset in prod, and/or same-origin proxy under `frontend/app/api/**`; document rebuild-per-host.
- Acceptance: serve frontend via LAN IP: network tab shows calls to the real backend host; no `localhost:8000` in bundle.

### UI-04 — P0 — `validated:false` result displayed as if published
- Evidence: `validated` typed at `page.tsx:24`, returned at `controller/consensus.go:63`, never rendered (:97-105).
- Impact: confidence <66 (rule `model/consensus.go:95`) still shows median + tx as authoritative; contradicts oracle trust story.
- Fix: if `!validated`, show a clear "non validé — non publié" state, visually distinct, no publication styling.
- Acceptance: force a low-confidence run: UI labels it unvalidated; golden run (confidence 100) shows validated badge.

### UI-05 — P1 — No refresh/polling/manual refresh: WhatsApp submissions invisible until F5
- Evidence: `page.tsx:42-44` one-shot effect; no `setInterval`/refresh button (grep confirms); refresh only after consensus (:55).
- Impact: live WhatsApp beats (B, F in demo.sh) never appear during a projected demo; operator must reload mid-pitch.
- Fix: poll `/api/state` every 3-5s (pause when tab hidden) plus a manual "Actualiser" button; keep last-good data on error.
- Acceptance: submit via curl/WhatsApp with page open: row appears within one poll interval, no reload.

### UI-06 — P1 — No loading/disabled state: double-clicks duplicate consensus, buttons feel dead
- Evidence: buttons `page.tsx:69-70` never disabled; no pending state in :46-62; each POST republishes (`consensus.go:58`).
- Impact: slow network => repeated clicks create duplicate publications/rewards and flickering results.
- Fix: `pending` state per action: disable buttons, show progress label/spinner, re-enable in `finally`.
- Acceptance: double-click "Lancer le consensus" on throttled network: exactly one POST in network tab.

### UI-07 — P1 — No abort/race handling: stale responses can overwrite newer state
- Evidence: no `AbortController` anywhere (grep); `refresh`, `runConsensus`, `ask` race freely (`page.tsx:37-62`).
- Impact: out-of-order responses resurrect old submissions/answers; setState after unmount warns in dev.
- Fix: AbortController per effect/action + request sequence guard; ignore superseded responses.
- Acceptance: rapid refresh/ask clicking with delayed backend: UI always shows the latest response only.

### UI-08 — P1 — `/api/ask` question hardcoded and unencoded; no user input
- Evidence: `page.tsx:59` literal spaces/`à`/`?` in URL (demo.sh:111 percent-encodes); backend ignores `q` (`ask.go:11`).
- Impact: demo cannot ask any other question; raw URL breaks with stricter proxies; answer can read as magic.
- Fix: editable input defaulting to the demo question, `encodeURIComponent`; note backend currently answers last publication.
- Acceptance: default click still returns the 187 quote; edited question is sent encoded (`%C3%A0` etc.).

### UI-09 — P1 — Non-422 HTTP errors treated as success responses
- Evidence: `page.tsx:50-54` only 422 branches; 500/`{error}` JSON or plain-text errors flow into `setConsensus`/`setAnswer`.
- Impact: median/confidence render blank/undefined while looking like a result; corrupted state is indistinguishable from success.
- Fix: strict `r.ok` + shape validation before setState; map every non-OK to the error surface with status code.
- Acceptance: mock 500 on consensus/ask: visible error with status, previous good result unchanged, no blank fields.

### UI-10 — P1 — Accessibility/semantics gaps: table, errors, live results
- Evidence: `page.tsx:71` error is plain red `<p>` (no `role=alert`); :74-82 table lacks caption/`scope`; no `aria-live` (:97-111); grep: zero `aria-`.
- Impact: screen readers miss errors, consensus, and assistant answers; table headers not associated with cells.
- Fix: `role=alert` for errors, `aria-live=polite` for consensus/answer, `<caption>` + `th scope=col`, labelled buttons.
- Acceptance: axe/voiceover pass: error, new consensus, and answer are announced; table navigable by header.

### UI-11 — P1 — Empty and loading states indistinguishable from failure
- Evidence: `state=null` and `submissions=[]` both render an empty table (`page.tsx:32,85`); no placeholder text at :73-95.
- Impact: before demo.sh runs, judges see a bare grid: cannot tell "loading", "no data", or "broken" apart.
- Fix: distinct loading, empty ("Aucune soumission — lancez demo.sh / en attente WhatsApp"), and error states.
- Acceptance: fresh backend shows the empty-state guidance; killing backend shows error state instead.

### UI-12 — P1 — Consensus audit trail dropped: outcomes, score_delta, channel not surfaced
- Evidence: API returns outcomes (`consensus.go:64`) and score_delta (`model.go:60-64`); types drop both (`page.tsx:7-29`); channel/tier untyped.
- Impact: operator can see D "excluded" but cannot show why (deviation, note, reward delta) or which channel/tier a node used.
- Fix: type and render outcomes/rewards after consensus (status, deviation, score_delta) and channel/tier columns or detail row.
- Acceptance: after golden run, D row/panel shows excluded status with proof 12 context and reward 0, A shows its reward delta.

### UI-13 — P2 — `proof_score: null` rendered as unexplained "auto"
- Evidence: `page.tsx:89` (`=== null ? "auto"`); semantics live only in backend (`model.go:16`, `submit.go:60-61` tier-3).
- Impact: judges cannot tell if "auto" means missing, unverified, or trusted; FAO row looks like a bug.
- Fix: label "n/a — source automatique (tier 3)" with tooltip/text; keep numeric scores for photo nodes.
- Acceptance: FAO row explains itself in French without hovering backend knowledge.

### UI-14 — P2 — Fraud/status conveyed by color only, raw English status strings
- Evidence: row color at `page.tsx:86`, raw `{s.status}` at :90; UI French elsewhere (`layout.tsx:10` lang fr).
- Impact: color-blind/projector washout hides the fraud beat; "accepted/excluded" breaks the French demo narration.
- Fix: text badges with icon/label (e.g. "Exclu — preuve 12"), FR mapping of statuses, color as reinforcement only.
- Acceptance: grayscale screenshot still identifies D as excluded and why.

### UI-15 — P2 — Tx/hash display: full-length, no copy, no truncation, raw label
- Evidence: `page.tsx:102-103` prints full `tx` and `aiReportHash` inline with an English code label.
- Impact: unbreakable hashes overflow on stage, cannot be copied/checked live, label is meaningless to judges.
- Fix: monospace truncated hash + copy button with feedback; FR label "Empreinte du rapport IA"; no fake explorer link (chain is simulated).
- Acceptance: click copy puts full tx on clipboard; layout unbroken at 360px and on projector width.

### UI-16 — P2 — Inline styles only: no responsive behavior, deprecated table attributes
- Evidence: all styling inline (`page.tsx:67,71,74,86`); `border/cellPadding` props (:74); `layout.tsx:11` only resets margin.
- Impact: 5-column table overflows phones/small windows; no media queries possible; maintenance cost per tweak.
- Fix: `frontend/app/globals.css` (or CSS module) imported by layout; scrollable table wrapper; min touch targets.
- Acceptance: 360px viewport: no page-level horizontal scroll, buttons usable; desktop rendering unchanged.

### UI-17 — P2 — Raw number/date formatting; publication freshness invisible
- Evidence: raw `{s.value}`, `{s.reward}`, median/confidence at `page.tsx:88-101`; Publication timestamp exists (`model.go:57`) but untyped/unshown.
- Impact: decimals render inconsistently, rewards lack units, and nobody can tell how old the displayed price is.
- Fix: `Intl.NumberFormat("fr-FR")`, explicit FCFA/kg + reward unit, show publication time from state (with UI-02).
- Acceptance: values show consistent FR formatting; latest publication displays its time.

### UI-18 — P2 — No test/lint/typecheck scripts guard the golden demo UI
- Evidence: `frontend/package.json:5-9` only dev/build/start; no test deps; strict TS exists (`tsconfig.json`) but no script invokes it standalone.
- Impact: any fix for UI-01..UI-17 can silently break the 187 demo rendering; regressions found live on stage.
- Fix: add `typecheck` (`tsc --noEmit`) + lint + one smoke test of golden render (187/100, D excluded). Needs scope exception (see below).
- Acceptance: `npm run typecheck` passes; smoke check fails if median/D-excluded rendering regresses.

## Ordered implementation plan

1. Survival first (P0): UI-01 error surface + UI-09 status handling (shared fetch helper), then UI-02 render publications.
2. Trust display (P0): UI-04 validated badge/state; keep golden path (confidence 100) visually unchanged except badge.
3. Deployment (P0): UI-03 decide proxy vs build-arg discipline; verify on a non-localhost host.
4. Live demo (P1): UI-05 polling + manual refresh, UI-06 pending/disabled, UI-07 abort/sequence guard.
5. Interaction (P1): UI-08 ask input/encoding, UI-11 loading/empty states, UI-12 outcomes/rewards audit trail.
6. Polish (P1/P2): UI-10 a11y, UI-14 badges, UI-13 proof label, UI-15 hash copy, UI-17 formatting, UI-16 CSS/responsive.
7. Guardrails (P2): UI-18 typecheck/smoke test; re-run `scripts/demo.sh` golden case end-to-end after each phase.

## Do not touch / frozen

- Frozen demo contract: median **187**, confidence **100**, node D **excluded** with proof score **12**;
  `/api/ask` quotes **187** and never estimates (demo.sh:105-117). UI must surface, never recompute, these values.
- Frontend fix scope: `frontend/app/**` only (page.tsx, layout.tsx, new CSS/proxy routes under `app/`).
  UI-03 proxy/config and UI-18 package.json scripts need an explicit owner-approved scope exception.
- NEVER modify backend handlers/models, `scripts/demo.sh`, or `spec.md` contracts to make the UI pass.
- Keep existing wire field names (`median`, `validated`, `aiReportHash`, `proof_score`, ...) exactly as typed from the API.
