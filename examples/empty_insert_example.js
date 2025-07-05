// Example: Inserting empty records with default values

const { fromUri } = require('redi/orm');

async function main() {
    // Create an in-memory SQLite database
    const db = fromUri('sqlite://:memory:');
    await db.connect();

    // Define schema with default values
    await db.loadSchema(`
        model User {
            id        Int      @id @default(autoincrement())
            name      String?
            status    String   @default("active")
            createdAt DateTime @default(now())
        }
    `);

    await db.syncSchemas();

    console.log('=== Empty Insert Example ===\n');

    // Insert a record with empty data - will use all default values
    console.log('1. Inserting with empty object {}:');
    const result1 = await db.models.User.create({
        data: {}
    });
    console.log('Created user:', result1);

    // Insert another record with only name specified
    console.log('\n2. Inserting with only name:');
    const result2 = await db.models.User.create({
        data: { name: 'John' }
    });
    console.log('Created user:', result2);

    // Query all users to see the results
    console.log('\n3. All users in database:');
    const users = await db.models.User.findMany();
    console.log(JSON.stringify(users, null, 2));

    await db.close();
}

main().catch(console.error);