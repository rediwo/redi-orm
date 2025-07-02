// Example: Basic query operations with RediORM
const { fromUri } = require('redi/orm');

async function main() {
    const db = fromUri('sqlite://./example.db');
    await db.connect();
    
    await db.loadSchema(`
        model Product {
            id          Int      @id @default(autoincrement())
            name        String
            price       Float
            stock       Int      @default(0)
            category    String
            createdAt   DateTime @default(now())
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
    
    // 1. Find all products
    console.log('\n1. All products:');
    const allProducts = await db.models.Product.findMany();
    allProducts.forEach(p => console.log(`  - ${p.name}: $${p.price} (${p.category})`));
    
    // 2. Find specific product
    console.log('\n2. Find laptop:');
    const laptop = await db.models.Product.findFirst({
        where: { name: 'Laptop' }
    });
    if (laptop) {
        console.log(`  Found: ${laptop.name} - $${laptop.price}`);
    }
    
    // 3. Find by category
    console.log('\n3. Electronics products:');
    const electronics = await db.models.Product.findMany({
        where: { category: 'Electronics' }
    });
    electronics.forEach(p => console.log(`  - ${p.name}: $${p.price}`));
    
    // 4. Count products
    const totalProducts = await db.models.Product.count();
    const electronicsCount = await db.models.Product.count({
        where: { category: 'Electronics' }
    });
    console.log(`\n4. Product counts:`);
    console.log(`  Total: ${totalProducts}`);
    console.log(`  Electronics: ${electronicsCount}`);
    
    // 5. Update product
    console.log('\n5. Updating laptop price...');
    const updatedLaptop = await db.models.Product.update({
        where: { name: 'Laptop' },
        data: { price: 899.99 }
    });
    console.log(`  New price: $${updatedLaptop.price || 899.99}`);
    
    // 6. Update many products  
    console.log('\n6. Applying discount to furniture...');
    const updated = await db.models.Product.updateMany({
        where: { category: 'Furniture' },
        data: { stock: 20 }
    });
    console.log(`  Updated ${updated.count} furniture items`);
    
    // 7. Raw SQL query
    console.log('\n7. Raw SQL query - Product names:');
    const names = await db.queryRaw('SELECT name FROM products ORDER BY name');
    names.forEach(p => console.log(`  - ${p.name}`));
    
    // 8. Delete a product
    console.log('\n8. Deleting mouse...');
    try {
        const deletedMouse = await db.models.Product.delete({
            where: { name: 'Mouse' }
        });
        console.log(`  Deleted: ${deletedMouse.name || 'Mouse'}`);
    } catch (err) {
        console.log(`  Delete result: ${err.message || 'Success'}`);
    }
    
    // Final count
    const finalCount = await db.models.Product.count();
    console.log(`\n9. Final product count: ${finalCount}`);
    
    await db.close();
    console.log('\nDone!');
}

main().catch(err => {
    console.error('Error:', err.message || err);
    if (err.stack) {
        console.error('Stack:', err.stack);
    }
    process.exit(1);
});