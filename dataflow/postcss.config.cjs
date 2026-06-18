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
                const resolved = `${resolvedVarName} `;

                childNode.type = 'word';
                childNode.value = resolved;
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

const config = {
  plugins: [
    propertyInjectPlugin(),
    colorMixVarResolverPlugin(),
    transformShortcutPlugin(),
    addSpaceForEmptyVarFallback(),
    require('postcss-media-minmax'),
    require('@csstools/postcss-oklab-function'),
    require('postcss-nesting'),
  ],
};

module.exports = config;
