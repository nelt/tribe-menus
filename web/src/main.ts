// Entry point of the front end (ADR 0004). The imports run first: app.ts defines the elements
// of the application, which Lit renders in a microtask, after this module. The format of the
// screen, set here, is thus on the root before their first render (ADR 0030, point 2).
import "./app.ts";
import { trackFormat } from "./layout.ts";

trackFormat(window);
