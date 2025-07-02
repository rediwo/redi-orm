// Simple demo for redi-orm run command
const { fromUri } = require('redi/orm');

async function main() {
    console.log('Starting RediORM demo...\n');
    
    // Connect to database
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    console.log('✓ Connected to database');
    
    // Load schema
    await db.loadSchema(`
        model Task {
            id          Int      @id @default(autoincrement())
            title       String
            completed   Boolean  @default(false)
            createdAt   DateTime @default(now())
        }
    `);
    console.log('✓ Schema loaded');
    
    // Auto-migrate
    await db.syncSchemas();
    console.log('✓ Database migrated\n');
    
    // Create some tasks
    console.log('Creating tasks...');
    const task1 = await db.models.Task.create({
        data: { title: 'Learn RediORM' }
    });
    console.log(`  Created: "${task1.title}" (ID: ${task1.id})`);
    
    const task2 = await db.models.Task.create({
        data: { title: 'Build an awesome app', completed: true }
    });
    console.log(`  Created: "${task2.title}" (ID: ${task2.id}, completed: ${task2.completed})`);
    
    const task3 = await db.models.Task.create({
        data: { title: 'Write documentation' }
    });
    console.log(`  Created: "${task3.title}" (ID: ${task3.id})\n`);
    
    // Count tasks
    const totalTasks = await db.models.Task.count();
    console.log(`Total tasks: ${totalTasks}`);
    
    const completedTasks = await db.models.Task.count({
        where: { completed: true }
    });
    console.log(`Completed tasks: ${completedTasks}`);
    
    const pendingTasks = await db.models.Task.count({
        where: { completed: false }
    });
    console.log(`Pending tasks: ${pendingTasks}\n`);
    
    // Update tasks
    console.log('Updating tasks...');
    const updateResult = await db.models.Task.updateMany({
        where: { completed: false },
        data: { completed: true }
    });
    console.log(`  Updated ${updateResult.count} task(s)\n`);
    
    // Delete a task
    console.log('Deleting task...');
    const deleteResult = await db.models.Task.deleteMany({
        where: { id: task3.id }
    });
    console.log(`  Deleted ${deleteResult.count} task(s)\n`);
    
    // Final count
    const finalCount = await db.models.Task.count();
    console.log(`Final task count: ${finalCount}`);
    
    await db.close();
    console.log('\n✓ Demo completed successfully!');
}

main().catch(err => {
    console.error('\n❌ Error:', err.message || err);
    process.exit(1);
});