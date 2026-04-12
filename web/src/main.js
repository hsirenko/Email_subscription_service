import { getApiBase } from "./env.js";
import { postSubscribe } from "./api/subscribe.js";
import { createMessenger, setNetStatus } from "./ui/messages.js";

const API_BASE = getApiBase();

const form = document.getElementById("sub-form");
const msg = document.getElementById("msg");
const netStatus = document.getElementById("net-status");
const messenger = createMessenger(msg);

if (!API_BASE) {
  messenger.set(
    "Set VITE_API_URL (Fly app URL, no trailing slash) in Vercel env and redeploy.",
    "warn",
  );
  setNetStatus(netStatus, "");
} else {
  setNetStatus(netStatus, API_BASE);
}

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  if (!API_BASE) {
    messenger.set(
      "Missing VITE_API_URL. Add it in Vercel → Settings → Environment Variables.",
      "warn",
    );
    return;
  }

  const fd = new FormData(form);
  const email = String(fd.get("email") || "").trim();
  const repo = String(fd.get("repo") || "").trim();

  messenger.set("Broadcasting to chain…", "");

  try {
    const { ok, status, statusText, text } = await postSubscribe(API_BASE, {
      email,
      repo,
    });

    if (ok) {
      messenger.set(
        "Signal accepted. Check your inbox and confirm the subscription link.",
        "ok",
      );
      form.reset();
      return;
    }

    if (status === 409) {
      messenger.set("Already subscribed (pending or active) for this repo.", "err");
      return;
    }
    if (status === 404) {
      messenger.set("Repository not found on GitHub (owner/repo).", "err");
      return;
    }
    if (status === 400) {
      messenger.set("Invalid email or repo format. Use owner/repo.", "err");
      return;
    }

    messenger.set(
      `Request failed (${status}): ${text || statusText}`,
      "err",
    );
  } catch (err) {
    messenger.set(
      err instanceof Error ? err.message : "Network error — check CORS and API URL.",
      "err",
    );
  }
});
