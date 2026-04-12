/**
 * POST /api/subscribe with JSON body (matches swagger + handlers).
 * @param {string} apiBase
 * @param {{ email: string, repo: string }} body
 * @returns {Promise<{ ok: boolean, status: number, statusText: string, text: string }>}
 */
export async function postSubscribe(apiBase, body) {
  const res = await fetch(`${apiBase}/api/subscribe`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  const text = await res.text();
  return {
    ok: res.ok,
    status: res.status,
    statusText: res.statusText,
    text,
  };
}
