import { computed, reactive } from "vue";
import { defineStore } from "pinia";

/** A single "line" of logging output created by a call to `log()`. */
export type LogLine = { date: Date, text: string, is: LogType };

/** Use a store for the logging lines in the Terminal. */
export const useTerminal = defineStore("Terminal", () => {

  const lines = reactive<LogLine[]>([]);
  let limit = 2000;

  /** Append a line to the log. */
  function log(message: string, type: LogType = LogType.Black) {
    // remove first element if limit reached
    if (lines.length >= limit) lines.shift();
    // insert new line at the end
    lines.push({
      date:   new Date(),
      text:   message,
      is:     type,
    });
  };

  // A couple aliases for logging with predefined colors.
  const warn    = (m: string) => log(m, LogType.Warning);
  const error   = (m: string) => log(m, LogType.Danger);
  const success = (m: string) => log(m, LogType.Success);
  const info    = (m: string) => log(m, LogType.Info);

  /** Get the last few lines of the log. */
  function tail(n: number) { return computed(() => lines.slice(-n)); };

  /** Clear all lines from the terminal. */
  function clear() { while (lines.pop() != undefined); };

  return {
    lines, limit, log, tail, clear,
    warn, error, success, info };
});

// https://bulma.io/documentation/helpers/color-helpers/
export enum LogType {
  White = "white",
  Black = "black",
  Grey = "grey",
  Light = "light",
  Dark = "dark",
  Primary = "primary",
  Link = "link",
  Info = "info",
  Success = "success",
  Warning = "warning",
  Danger = "danger"
};