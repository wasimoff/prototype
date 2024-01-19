import type { Plugin, InjectionKey } from "vue";
import { ref } from "vue";

// Example of how to use a global property in the Vue instance, provided as a plugin
// to enable easy registration with `app.use(...)` and including Typescript types.
// For most purposes, you should probably just use a store.
class Thing {

  private i = ref(0);

  get count() { return this.i.value; }

  increment() {
    this.i.value++;
    console.log("incremented thing: { i: %d }", this.count);
  }

}

declare module "vue" {
  interface ComponentCustomProperties {
    $thing: Thing,
  }
}

// symbol key for typed injection
// https://vuejs.org/guide/typescript/composition-api.html#typing-provide-inject
export const thing = Symbol() as InjectionKey<Thing>;

export const Something: Plugin = (app, options) => {

  const theThing = new Thing();

  // register the $thing on globalProperties according to the above declaration
  app.config.globalProperties.$thing = theThing;

  // but also provide it in the instance for use in SFC <script setup> blocks,
  // which don't have a `this` and can't use `getCurrentInstance()` in methods
  app.provide(thing, theThing);

}

//* Usage:
//*   const $thing = inject(thing)!;
//*   function increment() {
//*     $thing.increment();
//*     terminal.log(`Incremented $thing: ${$thing.count}`, LogType.Warning);
//*   }