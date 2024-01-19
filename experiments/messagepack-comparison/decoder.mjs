import { decode } from "@msgpack/msgpack";
import { createInterface } from "readline";

// convert a base64 string to Uint8Array
const unbase = str => Buffer.from(str, "base64");

// use readline to ask for strings
const readline = createInterface({ input: process.stdin, output: process.stdout });
console.log("Enter base64-encoded MessagePacks here:");
readline.addListener("line", line => console.log(decode(unbase(line))));