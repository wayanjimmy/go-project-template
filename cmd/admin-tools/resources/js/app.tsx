import { createInertiaApp } from "@inertiajs/react";
import type { ComponentType } from "react";
import { createRoot } from "react-dom/client";
import "./app.css";

const pages = import.meta.glob<{ default: ComponentType }>("./pages/**/*.tsx", {
  eager: true,
});

createInertiaApp({
  resolve: (name) => pages[`./pages/${name}.tsx`],
  setup({ el, App, props }) {
    createRoot(el).render(<App {...props} />);
  },
  title: (title) => `${title} - Admin Tools`,
});
