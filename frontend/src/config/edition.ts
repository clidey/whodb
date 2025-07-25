/**
 * Edition configuration for determining which GraphQL types to use
 */

// Get edition from environment variable or default to 'ce'
export const BUILD_EDITION = (import.meta.env.VITE_BUILD_EDITION as 'ce' | 'ee') || 'ce';

// Helper to check if we're running in EE mode
export const isEnterpriseEdition = () => BUILD_EDITION === 'ee';

// Helper to get edition-specific paths
export const getEditionPath = (basePath: string) => {
  return `${basePath}/${BUILD_EDITION}`;
};