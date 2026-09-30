type FetchFn = typeof fetch;

let originalFetch: FetchFn | undefined;

export function stubFetch(impl: FetchFn) {
  if (!originalFetch) {
    originalFetch = globalThis.fetch.bind(globalThis);
  }
  globalThis.fetch = impl;
}

export function restoreFetch() {
  if (originalFetch) {
    globalThis.fetch = originalFetch;
    originalFetch = undefined;
  }
}
