import { Window } from "happy-dom";

const happyWindow = new Window({ url: "http://localhost/" });

for (const key of Object.getOwnPropertyNames(happyWindow)) {
  if (key === "undefined") continue;
  if (key in globalThis) continue;
  try {
    Object.defineProperty(globalThis, key, {
      configurable: true,
      writable: true,
      value: (happyWindow as unknown as Record<string, unknown>)[key],
    });
  } catch {
    // Some browser globals are read-only on the host runtime.
  }
}

Object.assign(globalThis, {
  window: happyWindow,
  self: globalThis,
  document: happyWindow.document,
  HTMLElement: happyWindow.HTMLElement,
  Node: happyWindow.Node,
  Element: happyWindow.Element,
  DocumentFragment: happyWindow.DocumentFragment,
  SVGElement: happyWindow.SVGElement,
  navigator: happyWindow.navigator,
  getComputedStyle: happyWindow.getComputedStyle.bind(happyWindow),
});

await import("@testing-library/jest-dom");
