import { Env } from "./worker_env";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "content-type": "application/json",
      "cache-control": "no-store",
    },
  });
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    if (request.method !== "GET") {
      return jsonResponse({ success: false, error: "Method not allowed" }, 405);
    }

    const url = new URL(request.url);
    if (url.pathname !== "/agent") {
      return jsonResponse({ success: false, error: "Not found" }, 404);
    }

    const providedToken =
      request.headers.get("x-secret-token") ||
      request.headers.get("x-installer-key");

    if (!providedToken) {
      return jsonResponse({ success: false, error: "Missing token" }, 401);
    }

    const validTokens = [env.SECRET_TOKEN, env.INSTALLER_KEY].filter(Boolean);

    if (!validTokens.includes(providedToken)) {
      return jsonResponse({ success: false, error: "Unauthorized" }, 401);
    }

    return jsonResponse(
      {
        SERVEREYE_API_URL: env.SERVEREYE_API_URL,
        USE_WORKER_API: env.USE_WORKER_API,
        AGENT_INSTALLER_KEY: env.INSTALLER_KEY,
      },
      200
    );
  },
};
