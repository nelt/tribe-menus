// Entry point of the front end (ADR 0004): sets the format of the screen, then defines the
// elements of the application, which render after this module (ADR 0030, point 2).
import "./app.ts";
import { trackFormat } from "./layout.ts";

trackFormat(window);
