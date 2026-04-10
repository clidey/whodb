import yaml from 'js-yaml';
import type { Plugin } from 'vite';

/**
 * Vite plugin that transforms YAML imports into pre-parsed JSON objects.
 *
 * This moves YAML parsing from the browser (runtime) to the build step,
 * so js-yaml is only a dev dependency — not shipped to the client.
 */
export default function yamlPlugin(): Plugin {
    return {
        name: 'yaml-to-json',
        transform(code, id) {
            if (!id.endsWith('.yaml') && !id.endsWith('.yml')) return null;
            const parsed = yaml.load(code);
            return { code: `export default ${JSON.stringify(parsed)}`, map: null };
        },
    };
}
