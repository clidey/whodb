// Codemod: normalise `asChild` on @clidey/ux trigger elements.
//
//   node scripts/codemod-trigger-aschild.mjs <dir> [<dir>...] [--dry-run]
//
// - Removes an explicit `asChild` when the single child is a ux component the
//   library now infers for (Button, Badge, SidebarMenuButton, intrinsic
//   button/a/span/div, ...), since @clidey/ux >= 1.4.0 infers it.
// - Keeps `asChild` for children of unknown slot-safety (react-router Link,
//   custom components, other intrinsics).
// - Flags, without editing, every site whose child is not a single element or
//   is on the deny list, and lists for review sites that pass a bare expression
//   or wrap an unknown component without `asChild`.

import fs from 'node:fs';
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

const args = process.argv.slice(2);
const dryRun = args.includes('--dry-run');
const dirs = args.filter((arg) => !arg.startsWith('--'));
if (dirs.length === 0) {
  console.error('usage: codemod-trigger-aschild.mjs <dir> [<dir>...] [--dry-run]');
  process.exit(2);
}

const flagged = [];
const review = [];
let removed = 0;
let kept = 0;

function removeAttribute(text, attr) {
  // Remove the attribute together with the whitespace before it.
  let start = attr.getStart();
  while (start > 0 && /\s/.test(text[start - 1])) start -= 1;
  return text.slice(0, start) + text.slice(attr.getEnd());
}

for (const dir of dirs) {
  for (const file of listTsxFiles(dir)) {
    const source = parse(file);
    const edits = [];
    forEachTrigger(source, (element, imports) => {
      const trigger = tagName(element);
      const asChild = asChildAttribute(element);
      const where = location(source, element);
      const shape = classifyChildren(element);

      if (shape.kind !== 'element') {
        if (asChild && asChild.value !== false) {
          if (shape.kind === 'expression') {
            review.push(`${where} <${trigger} asChild> child is an expression; make sure it is always a single ref-forwarding element`);
          } else {
            flagged.push(`${where} <${trigger} asChild> child is ${shape.kind}; asChild cannot apply`);
          }
        }
        return;
      }

      const status = childSlotStatus(shape.child, imports);
      const childName = tagName(shape.child);
      if (status === 'denied') {
        flagged.push(`${where} <${trigger}> wraps ${childName}, which does not forward refs; wrap it in a <span>`);
        return;
      }
      if (asChild && asChild.value === undefined) {
        flagged.push(`${where} <${trigger}> has a non-literal asChild; make it literal or remove it`);
        return;
      }
      if (asChild && asChild.value === false) {
        return; // Documented escape hatch; leave untouched.
      }
      if (status === 'inferred') {
        if (asChild) edits.push(asChild.attr);
        return;
      }
      if (asChild) {
        kept += 1;
      } else {
        review.push(`${where} <${trigger}> wraps ${childName} without asChild; the library renders its own button around it`);
      }
    });

    if (edits.length > 0) {
      let text = source.getFullText();
      for (const attr of edits.sort((a, b) => b.getStart() - a.getStart())) {
        text = removeAttribute(text, attr);
      }
      removed += edits.length;
      if (!dryRun) fs.writeFileSync(file, text);
    }
  }
}

console.log(`asChild removed (now inferred): ${removed}${dryRun ? ' (dry run, nothing written)' : ''}`);
console.log(`asChild kept (unknown slot-safety): ${kept}`);
if (flagged.length > 0) {
  console.log(`\nFLAGGED (${flagged.length}) — fix by hand:`);
  for (const line of flagged) console.log(`  ${line}`);
}
if (review.length > 0) {
  console.log(`\nREVIEW (${review.length}) — check by hand:`);
  for (const line of review) console.log(`  ${line}`);
}
