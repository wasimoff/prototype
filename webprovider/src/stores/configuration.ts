import { defineStore } from "pinia";
import { computed, reactive } from "vue";

type Configuration = {
  // whether to connect to transport automatically
  autoconnect: boolean | null;
  // how many workers to start on launch; "nproc" means as many as there are cores
  workers: "max" | number | null;
  // broker transport URL and certificate hash
  transport: string | null;
  certhash: string | undefined | null;
  // endpoint for server config
  configpath: string | null;
}

// parse configuration from URL fragment and expose for application
export const useConfiguration = defineStore("Configuration", () => {

  // ---------- priority configuration via URL fragment --------- //

  // parse the URL fragment (the part after # symbol) as query parameters with URL class
  const fragments = new URLSearchParams(window.location.hash.substring(1));

  // get the relevant values from URL fragment
  const fragmentconf: Configuration = {
    autoconnect: asBoolean(fragments.get("autoconnect")),
    workers: (() => {
      let arg = fragments.get("workers");
      if (arg === null || arg === "max") return arg;
      let n = Number.parseInt(arg as string);
      if (Number.isNaN(n)) {
        console.error("is not a number:", arg);
        return 0;
      };
      return n;
    })(),
    transport: fragments.get("transport"),
    certhash: fragments.get("certhash"),
    configpath: fragments.get("config"),
  };


  // ---------- fetch configuration from server via http --------- //
  const serverconf: Configuration = reactive({
    // everything null by default until fetched
    autoconnect: null, workers: null, transport: null, certhash: null, configpath: null,
  });

  async function fetchConfig(path?: string) {
    // fetch from endpoint
    let response = await fetch(path || configpath);
    if (!response.ok) {
      console.error("can't fetch config:", response.status, response.statusText);
      throw new Error(`can't fetch configuration: ${response.status} ${response.statusText}`);
    };
    let json = await response.json();
    // set values from json, where it makes sense
    if (typeof json["transport"] === "string") serverconf.transport = json["transport"];
    if (typeof json["certhash"]  === "string") serverconf.certhash  = json["certhash"];
  }


  // ---------- default values --------- //
  const defaultconf: Configuration = {
    autoconnect: false,
    workers: "max",
    transport: "ws://localhost:4080/websocket/provider",
    certhash: undefined,
    configpath: window.location.origin + "/api/broker/v1/config",
  };


  // ---------- overall getters; mostly fragment > serverfetch > defaults --------- //
  const autoconnect = computed(() => firstOf(fragmentconf.autoconnect, defaultconf.autoconnect));
  const workers = computed(() => firstOf(fragmentconf.workers, defaultconf.workers));
  const transport = computed(() => firstOf(fragmentconf.transport, serverconf.transport, defaultconf.transport));
  const certhash = computed(() => firstOf(fragmentconf.certhash, serverconf.certhash, defaultconf.certhash));
  const configpath = firstOf(fragmentconf.configpath, defaultconf.configpath);


  return {
    fragmentconf, serverconf, defaultconf, fetchConfig,
    autoconnect, workers, transport, certhash, configpath,
  };
});

function firstOf<T>(...args: (T|null)[]): T {
  return args.find(arg => arg !== null) as T;
}

function asBoolean(s: string | null): boolean | null {
  if (s === null) return null;
  if (["", "true", "yes"].includes(s)) return true;
  return false;
}