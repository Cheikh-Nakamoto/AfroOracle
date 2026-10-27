const DEFAULT_BACKEND = "http://127.0.0.1:8000";

/** Forward same-origin dashboard and webhook requests to the private Go API. */
export async function proxyToBackend(request: Request, backendPath: string) {
  const base = process.env.AFRIORACLE_BACKEND_URL || DEFAULT_BACKEND;
  const incoming = new URL(request.url);
  const target = new URL(`${backendPath}${incoming.search}`, base);
  const headers = new Headers();
  const contentType = request.headers.get("content-type");
  if (contentType) headers.set("content-type", contentType);

  try {
    const upstream = await fetch(target, {
      method: request.method,
      headers,
      body: request.method === "GET" || request.method === "HEAD" ? undefined : await request.arrayBuffer(),
      cache: "no-store",
      signal: AbortSignal.timeout(30_000),
    });
    const responseHeaders = new Headers();
    const responseType = upstream.headers.get("content-type");
    if (responseType) responseHeaders.set("content-type", responseType);
    return new Response(await upstream.arrayBuffer(), { status: upstream.status, headers: responseHeaders });
  } catch {
    return Response.json({ error: "Le service AfriOracle est momentanément indisponible." }, { status: 502 });
  }
}
