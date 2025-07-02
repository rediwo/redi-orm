// Working demo for redi-orm run command
const { fromUri } = require('redi/orm');

async function main() {
    console.log('🚀 RediORM JavaScript Demo\n');
    
    // Connect to in-memory SQLite database
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    console.log('✓ Connected to database');
    
    // Define and load schema
    const schema = `
        model Product {
            id          Int      @id @default(autoincrement())
            name        String
            price       Float
            inStock     Boolean  @default(true)
            category    String
            createdAt   DateTime @default(now())
        }
        
        model Order {
            id          Int      @id @default(autoincrement())
            productId   Int
            quantity    Int
            total       Float
            createdAt   DateTime @default(now())
        }
    `;
    
    await db.loadSchema(schema);
    console.log('✓ Schema loaded');
    
    // Sync database schema
    await db.syncSchemas();
    console.log('✓ Database synced\n');
    
    // Create products
    console.log('📦 Creating products...');
    const products = [];
    
    products.push(await db.models.Product.create({
        data: { name: 'Laptop', price: 999.99, category: 'Electronics' }
    }));
    console.log(`  ✓ ${products[0].name} - $${products[0].price}`);
    
    products.push(await db.models.Product.create({
        data: { name: 'Mouse', price: 29.99, category: 'Electronics' }
    }));
    console.log(`  ✓ ${products[1].name} - $${products[1].price}`);
    
    products.push(await db.models.Product.create({
        data: { name: 'Desk', price: 199.99, category: 'Furniture', inStock: false }
    }));
    console.log(`  ✓ ${products[2].name} - $${products[2].price} (out of stock)`);
    
    // Count products
    console.log('\n📊 Product statistics:');
    const totalProducts = await db.models.Product.count();
    console.log(`  Total products: ${totalProducts}`);
    
    const inStockCount = await db.models.Product.count({
        where: { inStock: true }
    });
    console.log(`  In stock: ${inStockCount}`);
    
    const electronicsCount = await db.models.Product.count({
        where: { category: 'Electronics' }
    });
    console.log(`  Electronics: ${electronicsCount}`);
    
    // Create orders
    console.log('\n🛒 Creating orders...');
    const order1 = await db.models.Order.create({
        data: {
            productId: products[0].id,
            quantity: 1,
            total: products[0].price
        }
    });
    console.log(`  ✓ Order #${order1.id}: 1x ${products[0].name} = $${order1.total}`);
    
    const order2 = await db.models.Order.create({
        data: {
            productId: products[1].id,
            quantity: 3,
            total: products[1].price * 3
        }
    });
    console.log(`  ✓ Order #${order2.id}: 3x ${products[1].name} = $${order2.total}`);
    
    // Count orders
    const totalOrders = await db.models.Order.count();
    console.log(`\n📈 Total orders: ${totalOrders}`);
    
    // Use transaction for bulk operations
    console.log('\n💰 Processing bulk order in transaction...');
    await db.transaction(async (tx) => {
        // Create multiple orders in a transaction
        const bulkOrder1 = await tx.models.Order.create({
            data: {
                productId: products[0].id,
                quantity: 2,
                total: products[0].price * 2
            }
        });
        
        const bulkOrder2 = await tx.models.Order.create({
            data: {
                productId: products[2].id,
                quantity: 1,
                total: products[2].price
            }
        });
        
        console.log(`  ✓ Created ${2} orders in transaction`);
    });
    
    // Final count
    const finalOrderCount = await db.models.Order.count();
    console.log(`  Final order count: ${finalOrderCount}`);
    
    // Close database
    await db.close();
    console.log('\n✅ Demo completed successfully!');
    console.log('   The redi-orm run command is working! 🎉');
}

// Run the demo
main().catch(err => {
    console.error('\n❌ Error:', err.message || err);
    console.error('Stack:', err.stack);
    process.exit(1);
});