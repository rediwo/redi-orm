/**
 * Advanced Data Analysis with MCP
 * 
 * This example demonstrates advanced MCP features including:
 * - Batch queries
 * - Streaming large datasets
 * - Statistical analysis
 * - Data sampling
 * 
 * Start the MCP server:
 * redi-orm mcp --db=sqlite://./analytics.db --transport=http --port=3000
 */

const MCP_URL = 'http://localhost:3000/';
let requestId = 1;

async function mcpCall(method, params = {}) {
  const response = await fetch(MCP_URL, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      jsonrpc: '2.0',
      method,
      params,
      id: requestId++
    })
  });
  const result = await response.json();
  if (result.error) throw new Error(result.error.message);
  return result.result;
}

async function callTool(name, args = {}) {
  return mcpCall('tools/call', { name, arguments: args });
}

// Setup sample database with sales data
async function setupDatabase() {
  console.log('Setting up sample database...');
  
  // Create tables
  const createTables = [
    `CREATE TABLE IF NOT EXISTS sales (
      id INTEGER PRIMARY KEY,
      product_id INTEGER,
      customer_id INTEGER,
      amount REAL,
      quantity INTEGER,
      discount REAL DEFAULT 0,
      created_at TEXT DEFAULT CURRENT_TIMESTAMP,
      status TEXT DEFAULT 'pending'
    )`,
    `CREATE TABLE IF NOT EXISTS products (
      id INTEGER PRIMARY KEY,
      name TEXT,
      category TEXT,
      price REAL,
      cost REAL
    )`,
    `CREATE TABLE IF NOT EXISTS customers (
      id INTEGER PRIMARY KEY,
      name TEXT,
      email TEXT,
      country TEXT,
      joined_at TEXT DEFAULT CURRENT_TIMESTAMP
    )`
  ];

  for (const sql of createTables) {
    try {
      await callTool('query', { sql });
    } catch (e) {
      // Tables might already exist or we're in read-only mode
    }
  }
  
  console.log('✓ Database setup complete\n');
}

// Example 1: Batch Query Analysis
async function batchAnalysis() {
  console.log('=== 1. BATCH QUERY ANALYSIS ===\n');
  
  const batchResult = await callTool('batch_query', {
    queries: [
      {
        sql: "SELECT COUNT(*) as total_sales FROM sales",
        label: "total_count"
      },
      {
        sql: "SELECT SUM(amount) as revenue, AVG(amount) as avg_sale FROM sales WHERE status = 'completed'",
        label: "revenue_metrics"
      },
      {
        sql: `SELECT 
                strftime('%Y-%m', created_at) as month,
                COUNT(*) as sales_count,
                SUM(amount) as monthly_revenue
              FROM sales 
              WHERE status = 'completed'
              GROUP BY strftime('%Y-%m', created_at)
              ORDER BY month DESC
              LIMIT 12`,
        label: "monthly_trends"
      },
      {
        sql: `SELECT 
                p.category,
                COUNT(s.id) as sales_count,
                SUM(s.amount) as category_revenue
              FROM sales s
              JOIN products p ON s.product_id = p.id
              WHERE s.status = 'completed'
              GROUP BY p.category
              ORDER BY category_revenue DESC`,
        label: "category_performance"
      }
    ],
    fail_fast: false
  });

  const results = JSON.parse(batchResult.content[0].text);
  
  console.log(`Executed ${results.executed} queries:\n`);
  
  results.results.forEach(result => {
    console.log(`${result.label}:`);
    if (result.success) {
      console.log(`  Results: ${JSON.stringify(result.results, null, 2)}`);
    } else {
      console.log(`  Error: ${result.error}`);
    }
    console.log();
  });
}

// Example 2: Streaming Large Dataset
async function streamingExample() {
  console.log('=== 2. STREAMING LARGE DATASETS ===\n');
  
  const streamResult = await callTool('stream_query', {
    sql: `SELECT 
            s.id,
            s.amount,
            s.created_at,
            p.name as product_name,
            c.name as customer_name
          FROM sales s
          JOIN products p ON s.product_id = p.id
          JOIN customers c ON s.customer_id = c.id
          ORDER BY s.created_at DESC`,
    batch_size: 50
  });

  const stream = JSON.parse(streamResult.content[0].text);
  
  console.log(`Streaming results:`);
  console.log(`  Total rows: ${stream.total_rows}`);
  console.log(`  Batch size: ${stream.batch_size}`);
  console.log(`  Batch count: ${stream.batch_count}`);
  console.log();
  
  // Process first few batches
  stream.batches.slice(0, 2).forEach((batch, index) => {
    console.log(`  Batch ${index + 1}: ${batch.length} rows`);
    console.log(`  First row: ${JSON.stringify(batch[0])}`);
    console.log();
  });
}

// Example 3: Statistical Analysis
async function statisticalAnalysis() {
  console.log('=== 3. STATISTICAL ANALYSIS ===\n');
  
  const analysisResult = await callTool('analyze_table', {
    table: 'sales',
    columns: ['amount', 'quantity', 'discount'],
    sample_size: 1000
  });

  const analysis = JSON.parse(analysisResult.content[0].text);
  
  console.log(`Analysis of ${analysis.table} table:`);
  console.log(`  Total rows: ${analysis.total_rows}`);
  console.log(`  Sample size: ${analysis.sample_size}`);
  console.log();
  
  console.log('Column Statistics:');
  for (const [column, stats] of Object.entries(analysis.statistics)) {
    console.log(`\n  ${column}:`);
    console.log(`    Data type: ${stats.data_type}`);
    console.log(`    Null count: ${stats.null_count}`);
    console.log(`    Unique values: ${stats.unique_count}`);
    if (stats.min_value !== undefined) {
      console.log(`    Min value: ${stats.min_value}`);
      console.log(`    Max value: ${stats.max_value}`);
    }
    if (stats.sample_values && stats.sample_values.length > 0) {
      console.log(`    Sample values: ${stats.sample_values.slice(0, 5).join(', ')}`);
    }
  }
  console.log();
}

// Example 4: Smart Data Sampling
async function dataSampling() {
  console.log('=== 4. DATA SAMPLING ===\n');
  
  // Regular sampling
  console.log('Regular sample (first 5 records):');
  const regularSample = await callTool('generate_sample', {
    table: 'customers',
    count: 5,
    random: false
  });
  
  const regular = JSON.parse(regularSample.content[0].text);
  console.log(`  Retrieved ${regular.sample_size} samples`);
  regular.data.forEach(row => {
    console.log(`  - ${row.name} (${row.country})`);
  });
  console.log();
  
  // Random sampling
  console.log('Random sample (5 random customers):');
  const randomSample = await callTool('generate_sample', {
    table: 'customers',
    count: 5,
    random: true
  });
  
  const random = JSON.parse(randomSample.content[0].text);
  console.log(`  Retrieved ${random.sample_size} random samples`);
  random.data.forEach(row => {
    console.log(`  - ${row.name} (${row.country})`);
  });
  console.log();
  
  // Filtered sampling
  console.log('Filtered sample (US customers only):');
  const filteredSample = await callTool('generate_sample', {
    table: 'customers',
    count: 5,
    random: true,
    where: { country: 'USA' }
  });
  
  const filtered = JSON.parse(filteredSample.content[0].text);
  console.log(`  Retrieved ${filtered.sample_size} US customers`);
  filtered.data.forEach(row => {
    console.log(`  - ${row.name} (${row.country})`);
  });
}

// Example 5: Performance Analysis Dashboard
async function performanceDashboard() {
  console.log('\n=== 5. PERFORMANCE DASHBOARD ===\n');
  
  // Use batch queries for dashboard metrics
  const dashboardResult = await callTool('batch_query', {
    queries: [
      {
        sql: `SELECT 
                COUNT(DISTINCT customer_id) as unique_customers,
                COUNT(*) as total_orders,
                SUM(amount) as total_revenue,
                AVG(amount) as avg_order_value
              FROM sales 
              WHERE status = 'completed'`,
        label: "key_metrics"
      },
      {
        sql: `SELECT 
                strftime('%H', created_at) as hour,
                COUNT(*) as order_count
              FROM sales
              WHERE date(created_at) = date('now')
              GROUP BY hour
              ORDER BY hour`,
        label: "today_hourly"
      },
      {
        sql: `SELECT 
                c.country,
                COUNT(s.id) as orders,
                SUM(s.amount) as revenue
              FROM sales s
              JOIN customers c ON s.customer_id = c.id
              WHERE s.status = 'completed'
              GROUP BY c.country
              ORDER BY revenue DESC
              LIMIT 5`,
        label: "top_countries"
      }
    ]
  });

  const dashboard = JSON.parse(dashboardResult.content[0].text);
  
  dashboard.results.forEach(result => {
    if (result.success && result.results.length > 0) {
      console.log(`${result.label.toUpperCase()}:`);
      console.log(JSON.stringify(result.results, null, 2));
      console.log();
    }
  });
}

// Main execution
async function main() {
  try {
    await setupDatabase();
    
    // Run all examples
    await batchAnalysis();
    await streamingExample();
    await statisticalAnalysis();
    await dataSampling();
    await performanceDashboard();
    
    console.log('\n=== Advanced analysis examples completed! ===');
    
  } catch (error) {
    console.error('Error:', error.message);
  }
}

main().catch(console.error);