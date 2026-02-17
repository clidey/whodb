/**
 * Base path helpers for sub-path deployments.
 */

/**
 * Returns the normalized base path for the app.
 * Empty string means the app is mounted at the domain root.
 */
export const getBasePath = (): string => {
  const rawBase = (import.meta.env.BASE_URL || '/').trim();
  if (rawBase === '' || rawBase === '/') {
    return '';
  }
  return rawBase.endsWith('/') ? rawBase.slice(0, -1) : rawBase;
};

/**
 * Prefixes an absolute path with the configured base path.
 */
export const withBasePath = (path: string): string => {
  const basePath = getBasePath();
  const normalizedPath = path.startsWith('/') ? path : `/${path}`;
  if (basePath === '') {
    return normalizedPath;
  }
  if (normalizedPath === '/') {
    return basePath;
  }
  return `${basePath}${normalizedPath}`;
};
