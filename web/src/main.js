const raw = import.meta.env.VITE_API_URL;
const API_BASE = typeof raw === "string" ? raw.replace(/\/$/, "") : "";

const form = document.getElementById("sub-form");
const msg = document.getElementById("msg");
const netStatus = document.getElementById("net-status");

function setMsg(text, kind) {
  msg.textContent = text;
  msg.className = "msg" + (kind ? ` ${kind}` : "");
}

if (!API_BASE) {
  setMsg(
    "Set VITE_API_URL (Fly app URL, no trailing slash) in Vercel env and redeploy.",
    "warn",
  );
  netStatus.textContent = "API: not configured";
} else {
  netStatus.textContent = `API: ${API_BASE.replace(/^https?:\/\//, "")}`;
}

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  if (!API_BASE) {
    setMsg("Missing VITE_API_URL. Add it in Vercel → Settings → Environment Variables.", "warn");
    return;
  }

  const fd = new FormData(form);
  const email = String(fd.get("email") || "").trim();
  const repo = String(fd.get("repo") || "").trim();

  setMsg("Broadcasting to chain…", "");

  try {
    const res = await fetch(`${API_BASE}/api/subscribe`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, repo }),
    });

    const text = await res.text();

    if (res.ok) {
      setMsg(
        "Signal accepted. Check your inbox and confirm the subscription link.",
        "ok",
      );
      form.reset();
      return;
    }

    if (res.status === 409) {
      setMsg("Already subscribed (pending or active) for this repo.", "err");
      return;
    }
    if (res.status === 404) {
      setMsg("Repository not found on GitHub (owner/repo).", "err");
      return;
    }
    if (res.status === 400) {
      setMsg("Invalid email or repo format. Use owner/repo.", "err");
      return;
    }

    setMsg(`Request failed (${res.status}): ${text || res.statusText}`, "err");
  } catch (err) {
    setMsg(
      err instanceof Error ? err.message : "Network error — check CORS and API URL.",
      "err",
    );
  }
});
