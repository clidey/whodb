import { Constants } from "./common";

const [height, width] = [960, 1536];

function login() {
  cy.visit(`${Constants.Url}/login?host=localhost&username=user&password=password&database=Adventureworks`);
  cy.get('[data-testid="login-card"]', { timeout: 10000 }).should("be.visible");
  cy.wait(300);
  cy.get('button:nth-child(2)').click();
  cy.wait(300);
}

function changeSchema(index: number) {
  cy.get('[data-testid="schema-selector"]').click();
  cy.get(`[data-testid="schema-selector"] li:nth-child(${index})`).click();
  cy.wait(300);
}

describe("screenshots", () => {
  beforeEach(async () => {
    cy.viewport(width, height);
    cy.window().then((win) => {
      const style = win.document.createElement('style');
      style.type = 'text/css';
      const css = 'ul[data-testid="notifications"] { display: none; }';
      style.textContent = css;
      win.document.head.appendChild(style);
    });
  });

  it("should screenshot login page", () => {
    cy.visit(Constants.Url);
    cy.get('[data-testid="login-card"]', { timeout: 10000 }).should("be.visible");
    cy.wait(300);
    const [midX, midY] = [(width / 2), (height / 2)];
    cy.screenshot("login", {
      overwrite: true,
      clip: {
        x: midX-400,
        y: midY-350,
        height: 580,
        width: 600,
      },
    });
  });

  it("should screenshot tables page", () => {
    login();
    changeSchema(8);
    cy.screenshot("tables", {
      overwrite: true,
      clip: {
        x: 0,
        y: 0,
        height: 600,
        width: 1200,
      },
    });
  });

  it("should screenshot graph page", () => {
    login();
    changeSchema(8);
    cy.get('[data-testid="sidebar-navigation"] > div:nth-child(3)').click();
    cy.wait(1000);
    cy.screenshot("graph", {
      overwrite: true,
      clip: {
        x: 300,
        y: 0,
        height,
        width,
      },
    });
  });
});