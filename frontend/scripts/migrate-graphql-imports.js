#!/usr/bin/env node

/**
 * Migration script to update GraphQL imports to use the new @graphql alias
 * This allows for conditional CE/EE imports based on build edition
 * 
 * Usage: node scripts/migrate-graphql-imports.js
 */

const fs = require('fs');
const path = require('path');

// Pattern to match existing GraphQL imports
const importPatterns = [
  /from\s+['"]\.\.\/generated\/graphql['"]/g,
  /from\s+['"]\.\.\/\.\.\/generated\/graphql['"]/g,
  /from\s+['"]\.\/generated\/graphql['"]/g,
];

// Replacement string
const replacement = 'from \'@graphql\'';

// Recursively find all TypeScript/JavaScript files
function findFiles(dir, fileList = []) {
  const files = fs.readdirSync(dir);
  
  files.forEach(file => {
    const filePath = path.join(dir, file);
    const stat = fs.statSync(filePath);
    
    if (stat.isDirectory()) {
      // Skip node_modules and generated directories
      if (file !== 'node_modules' && file !== 'generated') {
        findFiles(filePath, fileList);
      }
    } else if (file.match(/\.(ts|tsx|js|jsx)$/)) {
      fileList.push(filePath);
    }
  });
  
  return fileList;
}

// Find all TypeScript/JavaScript files
const files = findFiles('src');

console.log(`Found ${files.length} files to check...`);

let updatedCount = 0;

files.forEach(file => {
  const filePath = path.resolve(file);
  let content = fs.readFileSync(filePath, 'utf8');
  let hasChanges = false;

  importPatterns.forEach(pattern => {
    if (pattern.test(content)) {
      content = content.replace(pattern, replacement);
      hasChanges = true;
    }
  });

  if (hasChanges) {
    fs.writeFileSync(filePath, content);
    console.log(`✓ Updated: ${file}`);
    updatedCount++;
  }
});

console.log(`\nMigration complete! Updated ${updatedCount} files.`);
console.log('\nNext steps:');
console.log('1. Run "npm run codegen:ce" to generate CE GraphQL types');
console.log('2. For EE builds, run "npm run codegen:ee" to generate EE GraphQL types');
console.log('3. Build with VITE_BUILD_EDITION=ce or VITE_BUILD_EDITION=ee as needed');