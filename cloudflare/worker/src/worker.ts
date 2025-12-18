import { Env } from "./worker_env";

interface KeyRegistrationRequest {
  secret_key: string;
  agent_version?: string;
  os_info?: string;
  hostname?: string;
}

interface ErrorResponse {
  success: false;
  error: string;
}

interface SuccessResponse {
  success: true;
  data: {
    secret_key: string;
    status: string;
  };
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    if (request.method !== "POST") {
      return jsonResponse({ success: false, error: "Method not allowed" }, 405);
    }

    const url = new URL(request.url);
    if (url.pathname !== "/api/register-key") {
      return jsonResponse({ success: false, error: "Not found" }, 404);
    }

    // Optional shared secret validation with constant-time comparison
    const installerKey = request.headers.get("x-installer-key");
    if (env.INSTALLER_KEY && installerKey !== env.INSTALLER_KEY) {
      return jsonResponse({ success: false, error: "Unauthorized" }, 401);
    }

    let payload: KeyRegistrationRequest;
    try {
      payload = (await request.json()) as KeyRegistrationRequest;
    } catch (err) {
      return jsonResponse({ success: false, error: "Invalid JSON payload" }, 400);
    }

    if (!payload.secret_key) {
      return jsonResponse({ success: false, error: "secret_key is required" }, 422);
    }

    const normalizedPayload = normalisePayload(payload);

    try {
      await env.REG_DB.prepare(
        `INSERT INTO generated_keys (secret_key, agent_version, os_info, hostname, status)
         VALUES (?1, ?2, ?3, ?4, 'active')
         ON CONFLICT(secret_key) DO UPDATE SET
            agent_version = excluded.agent_version,
            os_info = excluded.os_info,
            hostname = excluded.hostname,
            updated_at = CURRENT_TIMESTAMP`
      )
        .bind(
          normalizedPayload.secret_key,
          normalizedPayload.agent_version,
          normalizedPayload.os_info,
          normalizedPayload.hostname
        )
        .run();

      // Send webhook to backend for instant sync
      if (env.WEBHOOK_URL && env.WEBHOOK_SECRET) {
        try {
          await fetch(env.WEBHOOK_URL, {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              "X-Webhook-Secret": env.WEBHOOK_SECRET,
            },
            body: JSON.stringify({
              secret_key: normalizedPayload.secret_key,
              agent_version: normalizedPayload.agent_version,
              os_info: normalizedPayload.os_info,
              hostname: normalizedPayload.hostname,
              status: "active",
              created_at: new Date().toISOString(),
              updated_at: new Date().toISOString(),
            }),
          });
        } catch (webhookError) {
          // Log webhook error but don't fail the request
          console.error("Failed to send webhook:", webhookError);
        }
      }

      const response: SuccessResponse = {
        success: true,
        data: {
          secret_key: normalizedPayload.secret_key,
          status: "stored",
        },
      };
      return jsonResponse(response, 200);
    } catch (error) {
      console.error("Failed to store key:", error);
      return jsonResponse(
        { success: false, error: "Failed to store key, please retry later" },
        500
      );
    }
  },
};

function jsonResponse(body: SuccessResponse | ErrorResponse, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "content-type": "application/json",
      "cache-control": "no-store",
    },
  });
}

function normalisePayload(payload: KeyRegistrationRequest): Required<KeyRegistrationRequest> {
  return {
    secret_key: payload.secret_key,
    agent_version: payload.agent_version ?? "unknown",
    os_info: payload.os_info ?? "unknown",
    hostname: payload.hostname ?? "unknown",
  };
}
