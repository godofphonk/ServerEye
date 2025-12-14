/// <reference types="@cloudflare/workers-types" />

export interface Env {
  REG_DB: D1Database;
  INSTALLER_KEY?: string;
  WEBHOOK_URL?: string;
  WEBHOOK_SECRET?: string;
}
