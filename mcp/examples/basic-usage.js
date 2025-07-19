/**
 * Basic MCP Usage Example
 * 
 * This example demonstrates basic MCP operations using the HTTP transport.
 * Make sure to start the MCP server first:
 * 
 * redi-orm mcp --db=sqlite://./example.db --transport=http --port=3000
 */

const MCP_URL = 'http://localhost:3000/';
let requestId = 1;

// Helper function to make MCP calls
async function mcpCall(method, params = {}) {
  const response = await fetch(MCP_URL, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      jsonrpc: '2.0',
      method: method,
      params: params,
      id: requestId++
    })
  });

  const result = await response.json();
  
  if (result.error) {
    throw new Error(`MCP Error: ${result.error.message}`);
  }
  
  return result.result;
}

// Helper to call tools
async function callTool(toolName, args = {}) {
  return mcpCall('tools/call', {
    name: toolName,
    arguments: args
  });
}

async function main() {
  try {
    console.log('=== MCP Basic Usage Example ===\n');

    // 1. Initialize connection
    console.log('1. Initializing MCP connection...');
    const initResult = await mcpCall('initialize', {
      protocolVersion: '2024-11-05',
      capabilities: {},
      clientInfo: {
        name: 'example-client',
        version: '1.0.0'
      }
    });
    console.log('Server name:', initResult.serverInfo.name);
    console.log('Server version:', initResult.serverInfo.version);
    console.log();

    // 2. List available tools
    console.log('2. Available tools:');
    const tools = await mcpCall('tools/list');
    tools.tools.forEach(tool => {
      console.log(`  - ${tool.name}: ${tool.description}`);
    });
    console.log();

    // 3. List tables in the database
    console.log('3. Database tables:');
    const tablesResult = await callTool('list_tables');
    const tables = JSON.parse(tablesResult.content[0].text);
    console.log(`  Database type: ${tables.database_type}`);
    console.log(`  Tables found: ${tables.count}`);
    tables.tables.forEach(table => {
      console.log(`  - ${table}`);
    });
    console.log();

    // 4. Create a sample table if it doesn't exist
    console.log('4. Creating sample table...');
    try {
      await callTool('query', {
        sql: `CREATE TABLE IF NOT EXISTS products (
          id INTEGER PRIMARY KEY,
          name TEXT NOT NULL,
          price REAL NOT NULL,
          category TEXT,
          in_stock BOOLEAN DEFAULT 1
        )`
      });
      console.log('  ✓ Table created');
    } catch (e) {
      console.log('  ! Table creation blocked (read-only mode)');
    }
    console.log();

    // 5. Inspect table schema
    console.log('5. Inspecting products table schema:');
    const schemaResult = await callTool('inspect_schema', {
      table: 'products'
    });
    const schema = JSON.parse(schemaResult.content[0].text);
    console.log('  Columns:');
    schema.columns.forEach(col => {
      console.log(`  - ${col.name} (${col.type})`);
    });
    console.log();

    // 6. Insert sample data (will fail in read-only mode)
    console.log('6. Attempting to insert sample data...');
    try {
      await callTool('query', {
        sql: `INSERT INTO products (name, price, category) VALUES 
              ('Laptop', 999.99, 'Electronics'),
              ('Coffee Maker', 79.99, 'Appliances'),
              ('Desk Chair', 299.99, 'Furniture')`
      });
      console.log('  ✓ Data inserted');
    } catch (e) {
      console.log('  ! Insert blocked (read-only mode)');
    }
    console.log();

    // 7. Query data
    console.log('7. Querying products:');
    const queryResult = await callTool('query', {
      sql: 'SELECT * FROM products WHERE price < ?',
      parameters: [500]
    });
    const data = JSON.parse(queryResult.content[0].text);
    console.log(`  Found ${data.count} products under $500`);
    if (data.results.length > 0) {
      console.log('  Sample results:');
      data.results.forEach(row => {
        console.log(`  - ${row.name}: $${row.price} (${row.category})`);
      });
    }
    console.log();

    // 8. Count records
    console.log('8. Counting products by category:');
    const countResult = await callTool('count_records', {
      table: 'products',
      where: { category: 'Electronics' }
    });
    const count = JSON.parse(countResult.content[0].text);
    console.log(`  Electronics products: ${count.count}`);
    console.log();

    // 9. List available resources
    console.log('9. Available resources:');
    const resources = await mcpCall('resources/list');
    console.log(`  Found ${resources.resources.length} resources`);
    resources.resources.slice(0, 5).forEach(resource => {
      console.log(`  - ${resource.uri} (${resource.name})`);
    });
    console.log();

    // 10. Read a resource
    console.log('10. Reading schema resource:');
    const schemaResource = await mcpCall('resources/read', {
      uri: 'schema://database'
    });
    const dbSchema = JSON.parse(schemaResource.contents[0].text);
    console.log(`  Database has ${Object.keys(dbSchema).length} tables`);
    console.log();

    console.log('=== Example completed successfully! ===');

  } catch (error) {
    console.error('Error:', error.message);
  }
}

// Run the example
main().catch(console.error);