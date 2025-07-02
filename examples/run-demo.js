// Demo script for redi-orm run command
const { fromUri } = require('redi/orm');

async function main() {
    console.log('🚀 RediORM Run Command Demo\n');
    
    // Connect to SQLite database
    const db = fromUri('sqlite://demo.db');
    await db.connect();
    console.log('✓ Connected to database');
    
    // Define schema
    const schema = `
        model Todo {
            id          Int      @id @default(autoincrement())
            title       String
            completed   Boolean  @default(false)
            priority    String   @default("medium")
            createdAt   DateTime @default(now())
        }
    `;
    
    await db.loadSchema(schema);
    console.log('✓ Schema loaded');
    
    await db.syncSchemas();
    console.log('✓ Database synced\n');
    
    // Create todos
    console.log('📝 Creating todos...');
    
    const todo1 = await db.models.Todo.create({
        data: { 
            title: 'Set up RediORM',
            completed: true,
            priority: 'high'
        }
    });
    console.log(`  ✓ "${todo1.title}" [${todo1.priority}] ${todo1.completed ? '✅' : '⬜'}`);
    
    const todo2 = await db.models.Todo.create({
        data: { 
            title: 'Write documentation',
            priority: 'high'
        }
    });
    console.log(`  ✓ "${todo2.title}" [${todo2.priority}] ${todo2.completed ? '✅' : '⬜'}`);
    
    const todo3 = await db.models.Todo.create({
        data: { 
            title: 'Add more features',
            priority: 'low'
        }
    });
    console.log(`  ✓ "${todo3.title}" [${todo3.priority}] ${todo3.completed ? '✅' : '⬜'}`);
    
    // Statistics
    console.log('\n📊 Todo Statistics:');
    
    const totalTodos = await db.models.Todo.count();
    console.log(`  Total todos: ${totalTodos}`);
    
    const completedCount = await db.models.Todo.count({
        where: { completed: true }
    });
    console.log(`  Completed: ${completedCount}`);
    
    const highPriorityCount = await db.models.Todo.count({
        where: { priority: 'high' }
    });
    console.log(`  High priority: ${highPriorityCount}`);
    
    // Query all todos
    console.log('\n📋 All Todos:');
    const allTodos = await db.models.Todo.findMany({
        orderBy: { createdAt: 'asc' }
    });
    
    allTodos.forEach((todo, index) => {
        const status = todo.completed ? '✅' : '⬜';
        const priority = todo.priority.padEnd(6);
        console.log(`  ${index + 1}. ${status} [${priority}] ${todo.title}`);
    });
    
    // Clean up
    await db.close();
    console.log('\n✅ Demo completed successfully!');
    console.log('   You can run this script with: ./redi-orm run examples/run-demo.js');
}

// Execute
main().catch(err => {
    console.error('\n❌ Error:', err.message || err);
    process.exit(1);
});