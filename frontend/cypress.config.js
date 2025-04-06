const { defineConfig } = require("cypress");
const { execSync } = require("child_process");

module.exports = defineConfig({
  e2e: {
    setupNodeEvents(on, config) {
      on('task', {
        execCommand(command) {
          try {
            const result = execSync(command, { stdio: 'inherit' });
            return { success: true, output: result.toString() };
          } catch (error) {
            return { success: false, error: error.toString() };
          }
        }
      });
    },
  },
});
