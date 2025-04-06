const dbHost = 'localhost';
const dbUser = 'user';
const dbPassword = 'password';

describe('Sqlite3 E2E test', () => {
  it('should login correctly', () => {
    // login and setup
    cy.login('Sqlite3', undefined, undefined, undefined, 'e2e_test.db');
    cy.selectDatabase("e2e_test.db");
    
    // get all tables
    cy.getTables().then(storageUnitNames => {
      cy.log(storageUnitNames);
      expect(storageUnitNames).to.be.an('array');
      expect(storageUnitNames).to.deep.equal([
        "order_items",
        "orders",
        "payments",
        "products",
        "users"
      ]);
    });

    // check users table and fields
    cy.explore("users");
    cy.getExploreFields().then(text => {
      const textLines = text.split("\n");
    
      const expectedPatterns = [
        /^users$/,
        /^Type: table$/,
        /^Count: .+$/, // Ignores actual size value
        /^id: INTEGER$/,
        /^username: TEXT$/,
        /^email: TEXT$/,
        /^password: TEXT$/,
        /^created_at: DATETIME$/
      ];
      expectedPatterns.forEach(pattern => {
        expect(textLines.some(line => pattern.test(line))).to.be.true;
      });
    });

    // check user default data
    cy.data("users");
    cy.sortBy(0);
    cy.getTableData().then(({ columns, rows }) => {
      expect(columns).to.deep.equal([
        "#",
        "id [INTEGER]",
        "username [TEXT]",
        "email [TEXT]",
        "password [TEXT]",
        "created_at [DATETIME]"
      ]);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "1",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1"
        ],
        [
            "2",
            "2",
            "jane_smith",
            "jane@example.com",
            "securepassword2"
        ],
        [
            "3",
            "3",
            "admin_user",
            "admin@example.com",
            "adminpass1"
        ]
      ]);
    });

    // check total count

    // check page size
    cy.setTablePageSize(1);
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "1",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1",
        ],
      ]);
    });

    // check conditions
    // todo: check all types
    cy.whereTable([
      ["id", "=", "3"],
    ]);
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
          "1",
          "3",
          "admin_user",
          "admin@example.com",
          "adminpass1",
        ]
      ]);
    });

    // check clearing of the query and page size
    cy.setTablePageSize(10);
    cy.clearWhereConditions();
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows.length).to.equal(3);
    });
    
    // todo: [NOT PASSING - FIX] check pagination on the bottom
    // cy.getPageNumbers().then(pageNumbers => expect(pageNumbers).to.deep.equal(['1']));
    
    // check editing capability
    cy.setTablePageSize(2);
    cy.submitTable();

    // test saving
    cy.updateRow(1, 2, "jane_smith1", false);
    cy.getTableData().then(({ rows }) => {
      expect(rows.slice(1).map(row => row.slice(0, -1))).to.deep.equal([
        [
          "",
          "2",
          "jane_smith1",
          "jane@example.com",
          "securepassword2",
        ]
      ]);
    });
    cy.updateRow(1, 2, "jane_smith", false);
    cy.getTableData().then(({ rows }) => {
      expect(rows.slice(1).map(row => row.slice(0, -1))).to.deep.equal([
        [
          "",
          "2",
          "jane_smith",
          "jane@example.com",
          "securepassword2",
        ]
      ]);
    });

    cy.updateRow(1, 2, "jane_smith");
    cy.getTableData().then(({ rows }) => {
      expect(rows.slice(1).map(row => row.slice(0, -1))).to.deep.equal([
        [
          "",
          "2",
          "jane_smith",
          "jane@example.com",
          "securepassword2",
        ]
      ]);
    });

    // check search
    cy.searchTable("john");
    cy.wait(250);
    cy.getHighlightedRows().then(rows => {
      expect(rows.length).to.equal(1);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "1",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1"
        ]
      ]);
    });

    // check graph
    cy.goto("graph");
    cy.getGraph().then(graph => {
      const expectedGraph = {
        "users": ["orders"],
        "orders": ["order_items", "payments"],
        "order_items": [],
        "products": ["order_items"],
        "payments": [],
      };
    
      Object.keys(expectedGraph).forEach(key => {
        expect(graph).to.have.property(key);
        expect(graph[key].sort()).to.deep.equal(expectedGraph[key].sort());
      });
    });
    cy.getGraphNode().then(text => {
      const textLines = text.split("\n");
      const expectedPatterns = [
        /^users$/,
        /^Type: table$/,
        /^Count: .+$/, // Ignores actual size value
        /^id: INTEGER$/,
        /^username: TEXT$/,
        /^email: TEXT$/,
      ];
      expectedPatterns.forEach(pattern => {
        expect(textLines.some(line => pattern.test(line))).to.be.true;
      });
    });

    // check sql query in scratchpad
    cy.goto("scratchpad");
    cy.writeCode(0, "SELECT * FROM users1;");
    cy.runCode(0);
    cy.getCellError(0).then(err => expect(err).to.equal('no such table: users1'));
    
    cy.writeCode(0, "SELECT * FROM users ORDER BY id;");
    cy.runCode(0);
    cy.getCellQueryOutput(0).then(({ rows, columns }) => {
      expect(columns).to.deep.equal([
        "#",
        "id [INTEGER]",
        "username [TEXT]",
        "email [TEXT]",
        "password [TEXT]",
        "created_at [DATETIME]"
      ]);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "1",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1"
        ],
        [
            "2",
            "2",
            "jane_smith",
            "jane@example.com",
            "securepassword2"
        ],
        [
            "3",
            "3",
            "admin_user",
            "admin@example.com",
            "adminpass1"
        ]
      ]);
    });

    cy.writeCode(0, "UPDATE users SET username='john_doe1' WHERE id=1");
    cy.runCode(0);
    cy.getCellActionOutput(0).then(output => expect(output).to.equal('Action Executed'));

    cy.writeCode(0, "UPDATE users SET username='john_doe' WHERE id=1");
    cy.runCode(0);
    cy.getCellActionOutput(0).then(output => expect(output).to.equal('Action Executed'));

    // add cell
    cy.addCell(0);
    cy.writeCode(1, "SELECT * FROM users WHERE id=1;");
    cy.runCode(1);
    cy.getCellQueryOutput(1).then(({ rows, columns }) => {
      expect(columns).to.deep.equal([
        "#",
        "id [INTEGER]",
        "username [TEXT]",
        "email [TEXT]",
        "password [TEXT]",
        "created_at [DATETIME]"
      ]);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "1",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1",
        ]
      ]);
    });

    // remove first cell
    cy.removeCell(0);

    // ensure the first cell has the second cell data
    cy.getCellQueryOutput(0).then(({ rows, columns }) => {
      expect(columns).to.deep.equal([
        "#",
        "id [INTEGER]",
        "username [TEXT]",
        "email [TEXT]",
        "password [TEXT]",
        "created_at [DATETIME]"
      ]);
      expect(rows.map(row => row.slice(0, -1))).to.deep.equal([
        [
            "1",
            "1",
            "john_doe",
            "john@example.com",
            "securepassword1",
        ]
      ]);
    });

    // logout
    cy.logout();
  });
});
