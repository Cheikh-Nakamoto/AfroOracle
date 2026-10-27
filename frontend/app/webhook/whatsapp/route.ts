import { proxyToBackend } from "../../proxy";

export async function GET(request: Request) {
  return proxyToBackend(request, "/webhook/whatsapp");
}

export async function POST(request: Request) {
  return proxyToBackend(request, "/webhook/whatsapp");
}
