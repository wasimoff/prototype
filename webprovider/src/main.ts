// use polyfills for browser quirks
import "@app/polyfills.ts";

// use global CSS base
import "@app/assets/main.css";

// use Bulma CSS for styling
// https://bulma.io/documentation/
import "bulma/css/bulma.css";

// check prerequisites before we do anything else
import { CheckFeatures } from "@wasimoff/func/featuredetection";
try { CheckFeatures() }
catch (err) { alert(err); }

// create the Vue app instance
import { createApp } from "vue";
import App from "@app/App.vue";
const app = createApp(App);

// register Pinia stores
import { createPinia } from "pinia";
app.use(createPinia());

// mount app in index.html
app.mount("#app");

// stub the pyodide loader to be able to use it in console
// TODO: remove me
import { loadPyodide } from "pyodide";
(globalThis as any)["loadPyodide"] = loadPyodide;
