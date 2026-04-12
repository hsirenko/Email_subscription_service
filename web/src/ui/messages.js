/**
 * Status line under the form (atoms: text + CSS modifier class).
 * @param {HTMLElement} el
 */
export function createMessenger(el) {
  return {
    /** @param {string} text @param {''|'ok'|'err'|'warn'} [kind] */
    set(text, kind) {
      el.textContent = text;
      el.className = "msg" + (kind ? ` ${kind}` : "");
    },
  };
}

/**
 * Header pill showing which API host the UI targets.
 * @param {HTMLElement} el
 * @param {string} apiBase
 */
export function setNetStatus(el, apiBase) {
  if (!apiBase) {
    el.textContent = "API: not configured";
    return;
  }
  el.textContent = `API: ${apiBase.replace(/^https?:\/\//, "")}`;
}
