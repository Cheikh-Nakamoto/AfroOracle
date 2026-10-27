import { createServer } from "node:http";

const submissions = [
  { node_id: "Agent A", tier: 2, rep_score: 88, value: 187, proof_score: 94, channel: "dashboard", status: "accepted", reward: 6.5, score_delta: 0 },
  { node_id: "Agent B", tier: 2, rep_score: 72, value: 191, proof_score: 91, channel: "whatsapp", status: "accepted", reward: 5, score_delta: 0 },
  { node_id: "Agent C", tier: 2, rep_score: 65, value: 185, proof_score: 89, channel: "dashboard", status: "accepted", reward: 5, score_delta: 0 },
  { node_id: "FAO", tier: 3, rep_score: 0, value: 190, proof_score: null, channel: "api", status: "accepted", reward: 0.5, score_delta: 0 },
  { node_id: "Agent D", tier: 2, rep_score: 61, value: 240, proof_score: 12, channel: "dashboard", status: "excluded", reward: 0, score_delta: -2 },
];

const publication = {
  median: 187,
  confidence: 100,
  validated: true,
  tx: "0x6783abcdef0123456789abcdef0123456789abcdef0123456789abcdefcb7d25",
  chain: "simulated",
  value_x100: 18700,
  aiReportHash: "0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
  timestamp: 1793112660,
};

const outcomes = [
  { NodeID: "Agent A", Status: "accepted", Note: "", Reward: 6.5, ScoreDelta: 0, Deviation: 0, Weight: 1.5 },
  { NodeID: "Agent B", Status: "accepted", Note: "", Reward: 5, ScoreDelta: 0, Deviation: 2.1, Weight: 1 },
  { NodeID: "Agent C", Status: "accepted", Note: "", Reward: 5, ScoreDelta: 0, Deviation: 1.1, Weight: 1 },
  { NodeID: "FAO", Status: "accepted", Note: "", Reward: 0.5, ScoreDelta: 0, Deviation: 1.6, Weight: 0.7 },
  { NodeID: "Agent D", Status: "excluded", Note: "preuve rejetée (score < 50)", Reward: 0, ScoreDelta: -2, Deviation: 28.3, Weight: 0.5 },
];

function sendJSON(response, status, body) {
  response.writeHead(status, { "content-type": "application/json" });
  response.end(JSON.stringify(body));
}

createServer(async (request, response) => {
  const url = new URL(request.url, "http://127.0.0.1:8111");
  if (url.pathname === "/health") {
    response.writeHead(200);
    response.end("ok");
    return;
  }
  if (url.pathname === "/api/state" && request.method === "GET") {
    sendJSON(response, 200, { submissions, publications: [publication], rewards: submissions.map(({ node_id, reward, score_delta }) => ({ node_id, reward, score_delta })) });
    return;
  }
  if (url.pathname === "/api/consensus" && request.method === "POST") {
    sendJSON(response, 200, { ...publication, outcomes });
    return;
  }
  if (url.pathname === "/api/ask" && request.method === "GET") {
    sendJSON(response, 200, { answer: `Prix de démonstration : 187 FCFA/kg. Question reçue : ${url.searchParams.get("q")}` });
    return;
  }
  if (url.pathname === "/api/submit" && request.method === "POST") {
    let body = "";
    for await (const chunk of request) body += chunk;
    const payload = JSON.parse(body || "{}");
    if (!payload.photo_b64 || !payload.node_id) {
      sendJSON(response, 400, { error: "photo_b64 et node_id requis" });
      return;
    }
    sendJSON(response, 201, { status: "accepted", submission: { NodeID: payload.node_id, Value: payload.value }, proof: { Score: 94, Model: "mock" } });
    return;
  }
  if (url.pathname === "/webhook/whatsapp" && request.method === "GET") {
    response.writeHead(200, { "content-type": "text/plain" });
    response.end(url.searchParams.get("hub.challenge") ?? "");
    return;
  }
  sendJSON(response, 404, { error: "not found" });
}).listen(8111, "127.0.0.1");
