// Lint guard: fail when a @clidey/ux trigger has `asChild` and its child cannot be slotted.
//
//   node scripts/check-trigger-children.mjs <dir> [<dir>...]
//
// Fails on `<XTrigger asChild>` whose child is a fragment, string literal,
// array, conditional expression, multiple children, or a component on the
// deny list in trigger-children.json. Deny-listed children fail even without
// `asChild`, since wrapping them is never what the author meant.

import {
  asChildAttribute,
  childSlotStatus,
  classifyChildren,
  forEachTrigger,
  listTsxFiles,
  location,
  parse,
  tagName,
} from './trigger-children-lib.mjs';

const dirs = process.argv.slice(2).filter((arg) => !arg.startsWith('--'));
if (dirs.length === 0) {
  console.error('usage: check-trigger-children.mjs <dir> [<dir>...]');
  process.exit(2);
}

const errors = [];
let triggers = 0;

for (const dir of dirs) {
  for (const file of listTsxFiles(dir)) {
    const source = parse(file);
    forEachTrigger(source, (element, imports) => {
      triggers += 1;
      const trigger = tagName(element);
      const asChild = asChildAttribute(element);
      const where = location(source, element);
      const shape = classifyChildren(element);

      if (shape.kind === 'element') {
        if (childSlotStatus(shape.child, imports) === 'denied') {
          errors.push(`${where} <${trigger}> wraps ${tagName(shape.child)}, which does not forward refs or spread props; wrap it in a <span>`);
        }
        return;
      }
      if (!asChild || asChild.value === false) return;
      // A bare identifier (`{node}`) may hold an element; the library validates it at runtime.
      if (shape.kind === 'expression') return;
      errors.push(`${where} <${trigger} asChild> child is ${shape.kind}; asChild needs exactly one element child`);
    });
  }
}

if (errors.length > 0) {
  console.error(`check-trigger-children: ${errors.length} problem(s) in ${triggers} trigger(s)`);
  for (const line of errors) console.error(`  ${line}`);
  console.error('See @clidey/ux docs/triggers-and-aschild.md and .agents/rules/react-frontend.md.');
  process.exit(1);
}
console.log(`check-trigger-children: ${triggers} trigger(s) OK`);
