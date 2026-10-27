import { proxyToBackend } from "../../proxy";

type Context = { params: { path: string[] } };
const API_PATHS = new Set(["state", "submit", "consensus", "ask"]);

function endpoint(context: Context) {
  const path = context.params.path.join("/");
  if (!API_PATHS.has(path)) return null;
  return `/api/${path}`;
}

export async function GET(request: Request, context: Context) {
  const path = endpoint(context);
  if (!path || !["state", "ask"].includes(context.params.path.join("/"))) {
    return Response.json({ error: "Route API introuvable." }, { status: 404 });
  }
  return proxyToBackend(request, path);
}

export async function POST(request: Request, context: Context) {
  const path = endpoint(context);
  if (!path || !["submit", "consensus"].includes(context.params.path.join("/"))) {
    return Response.json({ error: "Route API introuvable." }, { status: 404 });
  }
  return proxyToBackend(request, path);
}
