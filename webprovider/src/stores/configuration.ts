import { defineStore } from "pinia";
import { computed } from "vue";

type Configuration = {
  // whether to connect to transport automatically
  autoconnect: boolean | null;
  // how many workers to start on launch; "max" means as many as there are cores
  workers: number | null;
  // broker transport URL
  transport: string | null;
}

// parse configuration from URL fragment and expose for application
export const useConfiguration = defineStore("Configuration", () => {

  // ---------- default values --------- //

  const defaultconf: Configuration = {
    autoconnect: true,
    workers: navigator.hardwareConcurrency, // i.e. number of logical cores
    transport: window.location.origin,
  };


  // ---------- configuration via query parameters in URL fragment --------- //

  // parse the URL fragment (the part after # symbol) as query parameters with URL class
  const fragments = new URLSearchParams(window.location.hash.substring(1));

  // get the relevant values from URL fragment
  const fragmentconf: Configuration = {

    autoconnect: asBoolean(fragments.get("autoconnect")),

    workers: (() => {
      let arg = fragments.get("workers");
      if (arg === "max")
        return navigator.hardwareConcurrency;
      let n = Number.parseInt(arg as string);
      if (Number.isNaN(n))
        return null;
      return n;
    })(),

    transport: fragments.get("transport"),

  };

  // ---------- overall getters; mostly fragment > serverfetch > defaults --------- //
  const autoconnect = computed(() => firstOf(fragmentconf.autoconnect, defaultconf.autoconnect));
  const workers = computed(() => firstOf(fragmentconf.workers, defaultconf.workers));
  const transport = computed(() => firstOf(fragmentconf.transport, defaultconf.transport));

  return { fragmentconf, defaultconf, autoconnect, workers, transport };
});


// ---------- helpers for the fragment parsing --------- //

function firstOf<T>(...args: (T|null)[]): T {
  return args.find(arg => arg !== null) as T;
}

function asBoolean(s: string | null): boolean | null {
  if (s === null) return null;
  if (["", "true", "t", "yes", "y", "1", "on"].includes(s.toLowerCase())) return true;
  return false;
}
