// Example: Various query operations with RediORM
const { fromUri } = require('redi/orm');

async function main() {
    const db = fromUri('sqlite://./example.db');
    await db.connect();
    
    await db.loadSchema(`
        model Product {
            id          Int      @id @default(autoincrement())
            name        String
            description String?
            price       Float
            stock       Int      @default(0)
            category    String
            tags        String   @default("[]") // JSON array stored as string
            createdAt   DateTime @default(now())
            updatedAt   DateTime @updatedAt
        }
        
        model Order {
            id         Int        @id @default(autoincrement())
            items      OrderItem[]
            total      Float
            status     String     @default("pending")
            createdAt  DateTime   @default(now())
        }
        
        model OrderItem {
            id        Int     @id @default(autoincrement())
            order     Order   @relation(fields: [orderId], references: [id])
            orderId   Int
            product   Product @relation(fields: [productId], references: [id])
            productId Int
            quantity  Int
            price     Float
        }
    `);
    
    await db.syncSchemas();
    
    // Create sample products
    const products = await Promise.all([
        db.models.Product.create({ data: { name: 'Laptop', price: 999.99, stock: 10, category: 'Electronics' } }),
        db.models.Product.create({ data: { name: 'Mouse', price: 29.99, stock: 50, category: 'Electronics' } }),
        db.models.Product.create({ data: { name: 'Keyboard', price: 79.99, stock: 30, category: 'Electronics' } }),
        db.models.Product.create({ data: { name: 'Monitor', price: 299.99, stock: 15, category: 'Electronics' } }),
        db.models.Product.create({ data: { name: 'Desk', price: 199.99, stock: 5, category: 'Furniture' } }),
        db.models.Product.create({ data: { name: 'Chair', price: 149.99, stock: 8, category: 'Furniture' } }),
    ]);
    
    console.log('Created products:', products.length);
    
    // 1. Find with conditions
    console.log('\n1. Products under $100:');
    const affordable = await db.models.Product.findMany({
        where: { price: { lt: 100 } },
        orderBy: { price: 'asc' }
    });
    affordable.forEach(p => console.log(`  - ${p.name}: $${p.price}`));
    
    // 2. Complex conditions
    console.log('\n2. Electronics between $50-$500:');
    const midRange = await db.models.Product.findMany({
        where: {
            AND: [
                { category: 'Electronics' },
                { price: { gte: 50 } },
                { price: { lte: 500 } }
            ]
        }
    });
    midRange.forEach(p => console.log(`  - ${p.name}: $${p.price}`));
    
    // 3. Aggregations
    const stats = await db.models.Product.aggregate({
        _avg: { price: true },
        _max: { price: true },
        _min: { price: true },
        _count: true
    });
    console.log('\n3. Price statistics:', stats);
    
    // 4. Group by category
    console.log('\n4. Products by category:');
    const categories = await db.models.Product.groupBy({
        by: ['category'],
        _count: { category: true },
        _avg: { price: true }
    });
    categories.forEach(c => console.log(`  - ${c.category}: ${c._count.category} items, avg $${c._avg.price.toFixed(2)}`));
    
    // 5. Pagination
    console.log('\n5. Paginated results (page 1, 3 items):');
    const page1 = await db.models.Product.findMany({
        take: 3,
        skip: 0,
        orderBy: { name: 'asc' }
    });
    page1.forEach(p => console.log(`  - ${p.name}`));
    
    // 6. Update many
    const updated = await db.models.Product.updateMany({
        where: { stock: { lt: 10 } },
        data: { stock: { increment: 5 } }
    });
    console.log(`\n6. Updated ${updated.count} low-stock items`);
    
    // 7. Create order with items
    const order = await db.models.Order.create({
        data: {
            total: 1109.97,
            items: {
                create: [
                    { productId: products[0].id, quantity: 1, price: 999.99 },
                    { productId: products[1].id, quantity: 2, price: 29.99 },
                    { productId: products[2].id, quantity: 1, price: 79.99 }
                ]
            }
        },
        include: {
            items: {
                include: { product: true }
            }
        }
    });
    
    console.log('\n7. Created order:');
    console.log(`  Order #${order.id} - Total: $${order.total}`);
    order.items.forEach(item => {
        console.log(`    - ${item.quantity}x ${item.product.name} @ $${item.price}`);
    });
    
    // 8. Raw SQL query
    console.log('\n8. Raw SQL query - Top 3 expensive products:');
    const expensive = await db.queryRaw(
        'SELECT name, price FROM products ORDER BY price DESC LIMIT 3'
    );
    expensive.forEach(p => console.log(`  - ${p.name}: $${p.price}`));
    
    // 9. Distinct values
    console.log('\n9. Distinct categories:');
    const distinctCategories = await db.models.Product.findMany({
        distinct: ['category'],
        select: { category: true }
    });
    distinctCategories.forEach(c => console.log(`  - ${c.category}`));
    
    // 10. Delete operations
    const deleted = await db.models.OrderItem.deleteMany({
        where: { orderId: order.id }
    });
    console.log(`\n10. Cleaned up ${deleted.count} order items`);
    
    await db.close();
    console.log('\nDone!');
}

main().catch(err => {
    console.error('Error:', err);
    process.exit(1);
});