/** @type {import('postcss-load-config').Config} */
const postcss = require('postcss');
const valueParser = require('postcss-value-parser');

const propertyInjectPlugin = () => {
  return {
    Once(root) {
      const fallbackRules = [];

      root.walkAtRules('property', (rule) => {
        let varName = null;
        let initialValue = null;

        rule.walkDecls((decl) => {
          if (decl.prop === 'initial-value') {
            varName = rule.params.trim();
            initialValue = decl.value;
          }
        });

        if (varName && initialValue) {
          fallbackRules.push({ prop: varName, value: initialValue });
        }
      });

      if (fallbackRules.length > 0) {
        const fallbackAtRule = postcss.atRule({
          name: 'supports',
          params: 'not (background: paint(something))',
        });
        const fallbackRootRule = postcss.rule({ selector: ':root' });

        fallbackRules.forEach(({ prop, value }) => {
          fallbackRootRule.append({ prop, value });
        });

        fallbackAtRule.append(fallbackRootRule);

        let lastImportIndex = -1;
        root.nodes.forEach((node, i) => {
          if (node.type === 'atrule' && node.name === 'import') {
            lastImportIndex = i;
          }
        });

        if (lastImportIndex === -1) {
          root.prepend(fallbackAtRule);
        } else {
          root.insertAfter(root.nodes[lastImportIndex], fallbackAtRule);
        }
      }

      root.walkDecls((decl) => {
        if (!decl.value) return;

        decl.value = decl.value.replaceAll(/\bto\s+(left|right)\s+in\s+[\w-]+/g, (_, direction) => {
          return `to ${direction}`;
        });
      });
    },
    postcssPlugin: 'postcss-property-polyfill',
  };
};

propertyInjectPlugin.postcss = true;

const colorMixVarResolverPlugin = () => {
  return {
    Once(root) {
      const cssVariables = {};

      root.walkRules((rule) => {
        if (!rule.selectors) return;

        const isRootOrHost = rule.selectors.some((selector) => {
          return selector.includes(':root') || selector.includes(':host');
        });

        if (isRootOrHost) {
          rule.walkDecls((decl) => {
            if (decl.prop.startsWith('--')) {
              cssVariables[decl.prop] = decl.value.trim();
            }
          });
        }
      });

      root.walkDecls((decl) => {
        const originalValue = decl.value;
        if (!originalValue || !originalValue.includes('color-mix(')) return;

        const parsed = valueParser(originalValue);
        let modified = false;

        parsed.walk((node) => {
          if (node.type === 'function' && node.value === 'color-mix') {
            node.nodes.forEach((childNode) => {
              if (childNode.type === 'function' && childNode.value === 'var' && childNode.nodes.length > 0) {
                const varName = childNode.nodes[0]?.value;
                if (!varName) return;

                const resolvedVarName = cssVariables[varName] === undefined ? 'black' : cssVariables[varName];

                // No trailing space: it would produce `<color>  <pct>` (double space) inside
                // color-mix(), which @csstools/postcss-color-mix-function fails to parse, leaving
                // the color-mix() uncomputed.
                childNode.type = 'word';
                childNode.value = resolvedVarName;
                childNode.nodes = [];
                modified = true;
              }
            });
          }
        });

        if (modified) {
          decl.value = parsed.toString();
        }
      });
    },
    postcssPlugin: 'postcss-color-mix-var-resolver',
  };
};

colorMixVarResolverPlugin.postcss = true;

const transformShortcutPlugin = () => {
  return {
    Once(root) {
      const defaults = {
        rotate: [0, 0, 1, '0deg'],
        scale: [1, 1, 1],
        translate: [0, 0, 0],
      };

      const fallbackAtRule = postcss.atRule({
        name: 'supports',
        params: 'not (translate: 0)',
      });

      root.walkRules((rule) => {
        let hasTransformShorthand = false;
        const transformFunctions = [];

        rule.walkDecls((decl) => {
          if (/^(rotate|scale|translate)$/.test(decl.prop)) {
            hasTransformShorthand = true;

            const newValues = [...defaults[decl.prop]];
            const value = decl.value.replaceAll(/\)\s*var\(/g, ') var(');
            const userValues = postcss.list.space(value);

            if (decl.prop === 'rotate' && userValues.length === 1) {
              newValues.splice(-1, 1, ...userValues);
            } else {
              newValues.splice(0, userValues.length, ...userValues);
            }

            transformFunctions.push(`${decl.prop}3d(${newValues.join(',')})`);
          }
        });

        if (hasTransformShorthand && transformFunctions.length > 0) {
          const fallbackRule = postcss.rule({ selector: rule.selector });

          fallbackRule.append({
            prop: 'transform',
            value: transformFunctions.join(' '),
          });

          fallbackAtRule.append(fallbackRule);
        }
      });

      if (fallbackAtRule.nodes && fallbackAtRule.nodes.length > 0) {
        root.append(fallbackAtRule);
      }
    },
    postcssPlugin: 'postcss-transform-shortcut',
  };
};

transformShortcutPlugin.postcss = true;

const addSpaceForEmptyVarFallback = () => {
  return {
    OnceExit(root) {
      root.walkDecls((decl) => {
        if (!decl.value || !decl.value.includes('var(')) {
          return;
        }

        const parsed = valueParser(decl.value);
        let changed = false;

        parsed.walk((node) => {
          if (node.type === 'function' && node.value === 'var') {
            const commaIndex = node.nodes.findIndex((n) => {
              return n.type === 'div' && n.value === ',';
            });

            if (commaIndex === -1) return;

            const fallbackNodes = node.nodes.slice(commaIndex + 1);
            const fallbackText = fallbackNodes.map((n) => n.value).join('').trim();

            if (fallbackText === '') {
              const commaNode = node.nodes[commaIndex];
              if (commaNode.value === ',') {
                commaNode.value = ', ';
                changed = true;
              }
            }
          }
        });

        if (changed) {
          decl.value = parsed.toString();
        }
      });
    },
    postcssPlugin: 'postcss-add-space-for-empty-var-fallback',
  };
};

addSpaceForEmptyVarFallback.postcss = true;

// Tailwind v4 emits opacity-modified colors (e.g. `bg-warning/5`) as a solid `var(--color)`
// base declaration plus an `@supports (color: color-mix(...))` block holding the real
// translucent value. Browsers without color-mix() (Chrome < 111) fail the @supports test and
// fall back to the solid base, so the color renders at full opacity. Unwrap these guards so the
// resolved value applies unconditionally and wins by source order over the solid base. Runs as
// Once — before @csstools/postcss-color-mix-function below — because that plugin deliberately
// skips declarations nested inside a color-mix() feature query, so the guard must be gone first.
const flattenColorMixSupports = () => {
  return {
    postcssPlugin: 'postcss-flatten-color-mix-supports',
    Once(root) {
      root.walkAtRules('supports', (atRule) => {
        if (!atRule.params.includes('color-mix(')) {
          return;
        }

        if (atRule.nodes && atRule.nodes.length > 0) {
          atRule.replaceWith(...atRule.nodes);
        } else {
          atRule.remove();
        }
      });
    },
  };
};

flattenColorMixSupports.postcss = true;

const config = {
  plugins: [
    propertyInjectPlugin(),
    colorMixVarResolverPlugin(),
    // Unwrap Tailwind's `@supports (color-mix())` guards before computing color-mix() below.
    flattenColorMixSupports(),
    // Resolve color-mix() to a static rgb()/rgba(). Runs before postcss-oklab-function: oklab
    // splits each color into an rgb + color(display-p3) pair, and color-mix-function leaves the
    // display-p3 half as an uncomputed color-mix(), so the color-mix() must be collapsed first.
    require('@csstools/postcss-color-mix-function')({ preserve: false }),
    transformShortcutPlugin(),
    addSpaceForEmptyVarFallback(),
    require('postcss-media-minmax'),
    require('@csstools/postcss-oklab-function'),
    require('postcss-nesting'),
  ],
};

module.exports = config;
