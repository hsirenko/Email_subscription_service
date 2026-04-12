/**
 * Build-time API origin (Vite). Empty string means the UI should not call the API.
 * @returns {string}
 */
export function getApiBase() {
  const raw = import.meta.env.VITE_API_URL;
  return typeof raw === "string" ? raw.replace(/\/$/, "") : "";
}
