// Example: Long-running script to test timeout functionality
const { fromUri } = require('redi/orm');

async function main() {
    console.log('Starting long-running task...');
    
    const db = fromUri('sqlite://./example.db');
    await db.connect();
    
    await db.loadSchema(`
        model Task {
            id        Int      @id @default(autoincrement())
            name      String
            createdAt DateTime @default(now())
        }
    `);
    
    await db.syncSchemas();
    
    console.log('Creating tasks over time...');
    
    // Create tasks with delays
    for (let i = 1; i <= 10; i++) {
        await db.models.Task.create({
            data: { name: `Task ${i}` }
        });
        console.log(`Created task ${i} at ${new Date().toLocaleTimeString()}`);
        
        // Wait 2 seconds between each task
        await new Promise(resolve => setTimeout(resolve, 2000));
    }
    
    const count = await db.models.Task.count();
    console.log(`Total tasks created: ${count}`);
    
    await db.close();
    console.log('Done!');
}

main().catch(err => {
    console.error('Error:', err.message || err);
    process.exit(1);
});