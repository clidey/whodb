const dbHost = 'localhost';
const dbUser = 'user';
const dbPassword = 'password';

describe('Postgres E2E test', () => {
  it('should login correctly', () => {
    // login and setup
    cy.login('MongoDB', 'localhost', 'user', 'password');
    cy.selectSchema("test_db");
    
    // get all çollections
    cy.getTables().then(storageUnitNames => {
      cy.log(storageUnitNames);
      expect(storageUnitNames).to.be.an('array');
      expect(storageUnitNames).to.deep.equal([
        "order_items",
        "order_summary",
        "orders",
        "payments",
        "products",
        "system.views",
        "users",
      ]);
    });

    // check users table and fields
    cy.explore("users");
    cy.getExploreFields().then(text => {
      const textLines = text.split("\n");
    
      const expectedPatterns = [
        /^users$/,
        /^Type: Collection$/,
        /^Storage Size: .+$/, // Ignores actual size value
        /^Count: .+$/,      // Ignores actual count value
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
        "document [Document]"
      ]);
      expect(rows).to.deep.equal([
        [
            "1",
            "{\"_id\":\"67b9a9feb1ad17254ef3f54e\",\"created_at\":\"2025-02-22T10:42:06.577Z\",\"email\":\"john@example.com\",\"password\":\"securepassword1\",\"username\":\"john_doe\"}"
        ],
        [
            "2",
            "{\"_id\":\"67b9a9feb1ad17254ef3f54f\",\"created_at\":\"2025-02-22T10:42:06.577Z\",\"email\":\"jane@example.com\",\"password\":\"securepassword2\",\"username\":\"jane_smith\"}"
        ],
        [
            "3",
            "{\"_id\":\"67b9a9feb1ad17254ef3f550\",\"created_at\":\"2025-02-22T10:42:06.577Z\",\"email\":\"admin@example.com\",\"password\":\"adminpass\",\"username\":\"admin_user\"}"
        ]
      ]);
    });

    // check total count

    // check page size
    cy.setTablePageSize(1);
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows).to.deep.equal([
        [
            "1",
            "{\"_id\":\"67b9a9feb1ad17254ef3f54e\",\"created_at\":\"2025-02-22T10:42:06.577Z\",\"email\":\"john@example.com\",\"password\":\"securepassword1\",\"username\":\"john_doe\"}"
        ]
      ]);
    });


    // check conditions
    // todo: check all types
    cy.whereTable([
      ["_id", "eq", "67b9a9feb1ad17254ef3f550"],
    ]);
    cy.submitTable();
    cy.getTableData().then(({ rows }) => {
      expect(rows).to.deep.equal([
        [
            "1",
            "{\"_id\":\"67b9a9feb1ad17254ef3f54e\",\"created_at\":\"2025-02-22T10:42:06.577Z\",\"email\":\"john@example.com\",\"password\":\"securepassword1\",\"username\":\"john_doe\"}"
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
    cy.updateRow(1, 1, "{\"_id\":\"67b9a9feb1ad17254ef3f54f\",\"created_at\":\"2025-02-22T10:42:06.577Z\",\"email\":\"jane@example.com\",\"password\":\"securepassword2\",\"username\":\"jane_smith1\"}", false);    
    cy.getTableData().then(({ rows }) => {
      expect(rows.slice(1)).to.deep.equal([
        [
          "",
          "{\"_id\":\"67b9a9feb1ad17254ef3f54f\",\"created_at\":\"2025-02-22T10:42:06.577Z\",\"email\":\"jane@example.com\",\"password\":\"securepassword2\",\"username\":\"jane_smith1\"}"
        ]
      ]);
    });
    cy.updateRow(1, 1, "{\"_id\":\"67b9a9feb1ad17254ef3f54f\",\"created_at\":\"2025-02-22T10:42:06.577Z\",\"email\":\"jane@example.com\",\"password\":\"securepassword2\",\"username\":\"jane_smith\"}", false);
    cy.getTableData().then(({ rows }) => {
      expect(rows.slice(1)).to.deep.equal([
        [
          "",
          "{\"_id\":\"67b9a9feb1ad17254ef3f54f\",\"created_at\":\"2025-02-22T10:42:06.577Z\",\"email\":\"jane@example.com\",\"password\":\"securepassword2\",\"username\":\"jane_smith\"}",
        ]
      ]);
    });

    cy.updateRow(1, 1, "{\"_id\":\"67b9a9feb1ad17254ef3f54f\",\"created_at\":\"2025-02-22T10:42:06.577Z\",\"email\":\"jane@example.com\",\"password\":\"securepassword2\",\"username\":\"jane_smith1\"}");
    cy.getTableData().then(({ rows }) => {
      expect(rows.slice(1)).to.deep.equal([
        [
          "",
          "{\"_id\":\"67b9a9feb1ad17254ef3f54f\",\"created_at\":\"2025-02-22T10:42:06.577Z\",\"email\":\"jane@example.com\",\"password\":\"securepassword2\",\"username\":\"jane_smith\"}",
        ]
      ]);
    });

    // check search
    cy.searchTable("john");
    cy.wait(250);
    cy.getHighlightedRows().then(rows => {
      expect(rows.length).to.equal(1);
      expect(rows).to.deep.equal([
        [
            "1",
            "{\"_id\":\"67b9a9feb1ad17254ef3f54e\",\"created_at\":\"2025-02-22T10:42:06.577Z\",\"email\":\"john@example.com\",\"password\":\"securepassword1\",\"username\":\"john_doe\"}",
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
        "order_summary": []
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
        /^Type: Collection$/,
        /^Storage Size: .+$/, // Ignores actual size value
        /^Count: .+$/,      // Ignores actual count value
      ];
      expectedPatterns.forEach(pattern => {
        expect(textLines.some(line => pattern.test(line))).to.be.true;
      });
    });

    // logout
    cy.logout();
  });
});
