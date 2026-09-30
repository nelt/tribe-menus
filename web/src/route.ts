const tribePathPattern = /^\/tribes\/([^/]+)(?:\/|$)/;

// Returns the tribe identifier from a /tribes/<id>/… path, decoded,
// or undefined when the path is outside the application.
export function tribeIdFromPath(pathname: string): string | undefined {
  const encoded = tribePathPattern.exec(pathname)?.[1];
  if (encoded === undefined) {
    return undefined;
  }
  try {
    return decodeURIComponent(encoded);
  } catch {
    return undefined;
  }
}
