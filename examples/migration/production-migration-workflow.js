// Production Migration Workflow Example
// This demonstrates best practices for using file-based migrations in production

const { fromUri } = require('redi/orm');
const fs = require('fs');
const path = require('path');

// Helper to run shell commands
const { execSync } = require('child_process');

async function demonstrateProductionWorkflow() {
    console.log('=== Production Migration Workflow Demo ===\n');
    
    const baseDir = path.resolve('./tmp-workflow-demo');
    const migrationsDir = path.join(baseDir, 'migrations');
    const dbPath = path.join(baseDir, 'production.db');
    const dbUri = `sqlite://${dbPath}`;
    
    // Setup
    console.log('\n1. Setting up production environment...');
    if (fs.existsSync(baseDir)) {
        // Use execSync to remove directory
        execSync(`rm -rf ${baseDir}`);
    }
    fs.mkdirSync(baseDir, { recursive: true });
    fs.mkdirSync(migrationsDir, { recursive: true });
    
    // Step 1: Initial schema
    console.log('\n2. Creating initial schema...');
    const initialSchema = `
        model User {
            id        Int      @id @default(autoincrement())
            email     String   @unique
            name      String
            createdAt DateTime @default(now())
        }
        
        model Product {
            id          Int      @id @default(autoincrement())
            name        String
            description String?
            price       Float
            stock       Int      @default(0)
            createdAt   DateTime @default(now())
        }
    `;
    
    fs.writeFileSync(path.join(baseDir, 'schema.prisma'), initialSchema);
    console.log('✓ Initial schema created');
    
    // Generate initial migration
    console.log('\n3. Generating initial migration...');
    try {
        execSync(`../../redi-orm migrate:generate --db=${dbUri} --schema=${baseDir}/schema.prisma --migrations=${migrationsDir} --name="initial_setup"`, {
            stdio: 'inherit'
        });
    } catch (e) {
        console.log('Note: Migration generation requires the CLI to be built');
    }
    
    // Demonstrate the workflow with actual ORM operations
    console.log('\n4. Connecting to database and applying schema...');
    const db = fromUri(dbUri);
    await db.connect();
    
    // Load and sync initial schema
    await db.loadSchema(initialSchema);
    await db.syncSchemas();
    
    // Insert initial data
    console.log('\n5. Inserting initial production data...');
    const users = await Promise.all([
        db.models.User.create({ data: { email: 'admin@company.com', name: 'Admin User' } }),
        db.models.User.create({ data: { email: 'user1@company.com', name: 'John Doe' } }),
        db.models.User.create({ data: { email: 'user2@company.com', name: 'Jane Smith' } })
    ]);
    
    const products = await Promise.all([
        db.models.Product.create({ data: { name: 'Laptop', description: 'High-performance laptop', price: 999.99, stock: 50 } }),
        db.models.Product.create({ data: { name: 'Mouse', description: 'Wireless mouse', price: 29.99, stock: 200 } }),
        db.models.Product.create({ data: { name: 'Keyboard', description: 'Mechanical keyboard', price: 79.99, stock: 100 } })
    ]);
    
    console.log(`✓ Created ${users.length} users and ${products.length} products`);
    
    // Simulate schema evolution
    console.log('\n6. Evolving schema (adding Order model)...');
    const evolvedSchema = `
        model User {
            id        Int      @id @default(autoincrement())
            email     String   @unique
            name      String
            phone     String?  // New field
            orders    Order[]  // New relation
            createdAt DateTime @default(now())
            updatedAt DateTime @updatedAt  // New field
        }
        
        model Product {
            id          Int         @id @default(autoincrement())
            name        String
            description String?
            price       Float
            stock       Int         @default(0)
            sku         String?     // New field (nullable to handle existing data)
            orderItems  OrderItem[] // New relation
            createdAt   DateTime    @default(now())
            updatedAt   DateTime    @updatedAt  // New field
        }
        
        model Order {  // New model
            id         Int         @id @default(autoincrement())
            userId     Int
            user       User        @relation(fields: [userId], references: [id])
            items      OrderItem[]
            total      Float
            status     String      @default("pending")
            createdAt  DateTime    @default(now())
            updatedAt  DateTime    @updatedAt
        }
        
        model OrderItem {  // New model
            id        Int      @id @default(autoincrement())
            orderId   Int
            order     Order    @relation(fields: [orderId], references: [id])
            productId Int
            product   Product  @relation(fields: [productId], references: [id])
            quantity  Int
            price     Float
            
            @@unique([orderId, productId])
        }
    `;
    
    fs.writeFileSync(path.join(baseDir, 'schema-v2.prisma'), evolvedSchema);
    console.log('✓ Created evolved schema with Order management');
    
    // Close current connection before migration
    await db.close();
    
    // Demonstrate migration workflow steps
    console.log('\n7. Production Migration Best Practices:');
    console.log('   a) Generate migration file with descriptive name');
    console.log('   b) Review generated SQL before applying');
    console.log('   c) Test migration on staging environment first');
    console.log('   d) Backup production database before migration');
    console.log('   e) Apply migration during maintenance window');
    console.log('   f) Verify data integrity after migration');
    console.log('   g) Have rollback plan ready');
    
    // Reconnect with new schema
    console.log('\n8. Applying evolved schema...');
    const db2 = fromUri(dbUri);
    await db2.connect();
    await db2.loadSchema(evolvedSchema);
    await db2.syncSchemas();
    
    // Test new functionality
    console.log('\n9. Testing new Order functionality...');
    
    // Create an order
    const order = await db2.models.Order.create({
        data: {
            userId: users[0].id,
            total: 1079.97,
            status: 'confirmed'
        }
    });
    
    // Add order items
    await db2.models.OrderItem.create({
        data: {
            orderId: order.id,
            productId: products[0].id,
            quantity: 1,
            price: 999.99
        }
    });
    
    await db2.models.OrderItem.create({
        data: {
            orderId: order.id,
            productId: products[2].id,
            quantity: 1,
            price: 79.99
        }
    });
    
    console.log(`✓ Created order #${order.id} with 2 items`);
    
    // Verify data relationships
    console.log('\n10. Verifying data integrity...');
    const orderCount = await db2.models.Order.count();
    const itemCount = await db2.models.OrderItem.count();
    console.log(`✓ Orders: ${orderCount}, Order Items: ${itemCount}`);
    
    // Demonstrate rollback scenario
    console.log('\n11. Rollback Scenario:');
    console.log('    If issues are detected after migration:');
    console.log('    - Stop application servers');
    console.log('    - Run: redi-orm migrate:rollback --db=<uri> --migrations=<dir>');
    console.log('    - Restore application to previous version');
    console.log('    - Investigate and fix migration issues');
    console.log('    - Generate new migration with fixes');
    console.log('    - Test thoroughly before re-applying');
    
    await db2.close();
    
    // Migration checklist
    console.log('\n=== Production Migration Checklist ===');
    console.log('□ Schema changes reviewed by team');
    console.log('□ Migration tested on development environment');
    console.log('□ Migration tested on staging with production-like data');
    console.log('□ Database backup completed');
    console.log('□ Rollback procedure documented');
    console.log('□ Maintenance window scheduled');
    console.log('□ Monitoring alerts configured');
    console.log('□ Post-migration verification plan ready');
    
    console.log('\n✓ Demo completed!');
    console.log(`  Test files created in: ${baseDir}`);
}

// Error handling wrapper
async function main() {
    try {
        await demonstrateProductionWorkflow();
    } catch (error) {
        console.error('Error:', error && error.message ? error.message : error);
        if (error && error.stack) {
            console.error('Stack:', error.stack);
        }
        process.exit(1);
    }
}

main();