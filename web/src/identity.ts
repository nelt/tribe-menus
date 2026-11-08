// Identity files, read from docs/design/identite: a single source, no copy in web/src. The build
// names them after their content (ADR 0012, point 2) and gives their address here.
import logotype from "../../docs/design/identite/logotype.svg";
import symbol from "../../docs/design/identite/symbole.svg";
import reducedSymbol from "../../docs/design/identite/symbole-reduit.svg";

export const logotypeURL: string = logotype;
export const symbolURL: string = symbol;
/** The symbol for 48 px and less: the rail (docs/design/README.md, « Identité »). */
export const reducedSymbolURL: string = reducedSymbol;
