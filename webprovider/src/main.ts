// use polyfills for browser quirks
import "@/polyfills";

// use global CSS base
import "@/assets/main.css";

// use Bulma CSS for styling
// https://bulma.io/documentation/
import "bulma/css/bulma.css";

// check prerequisites before we do anything else
import { CheckFeatures } from "@/fn/featuredetection";
try { CheckFeatures() }
catch (err) { alert(err); }

// create the Vue app instance
import { createApp } from "vue";
import App from "@/App.vue";
const app = createApp(App);

// register Pinia stores
import { createPinia } from "pinia";
app.use(createPinia());

// mount app in index.html
app.mount("#app");
