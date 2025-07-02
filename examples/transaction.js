// Example: Transaction usage with RediORM
const { fromUri } = require('redi/orm');

async function main() {
    // Connect to database
    const db = fromUri('sqlite://./example.db');
    await db.connect();
    
    // Load schema
    await db.loadSchema(`
        model Account {
            id      Int    @id @default(autoincrement())
            name    String
            balance Float  @default(0)
        }
        
        model Transaction {
            id        Int      @id @default(autoincrement())
            from      Account  @relation("from", fields: [fromId], references: [id])
            fromId    Int
            to        Account  @relation("to", fields: [toId], references: [id])
            toId      Int
            amount    Float
            createdAt DateTime @default(now())
        }
    `);
    
    // Auto-migrate
    await db.syncSchemas();
    
    // Create two accounts
    const account1 = await db.models.Account.create({
        data: { name: 'Alice', balance: 1000 }
    });
    
    const account2 = await db.models.Account.create({
        data: { name: 'Bob', balance: 500 }
    });
    
    console.log('Initial balances:');
    console.log(`${account1.name}: $${account1.balance}`);
    console.log(`${account2.name}: $${account2.balance}`);
    
    // Transfer money using a transaction
    const transferAmount = 200;
    
    try {
        await db.transaction(async (tx) => {
            // Deduct from sender
            const sender = await tx.models.Account.update({
                where: { id: account1.id },
                data: { balance: account1.balance - transferAmount }
            });
            
            // Check if sender has sufficient balance
            if (sender.balance < 0) {
                throw new Error('Insufficient funds');
            }
            
            // Add to receiver
            await tx.models.Account.update({
                where: { id: account2.id },
                data: { balance: account2.balance + transferAmount }
            });
            
            // Record the transaction
            await tx.models.Transaction.create({
                data: {
                    fromId: account1.id,
                    toId: account2.id,
                    amount: transferAmount
                }
            });
            
            console.log(`\nTransferred $${transferAmount} from ${account1.name} to ${account2.name}`);
        });
        
        // Check final balances
        const finalAccount1 = await db.models.Account.findUnique({ where: { id: account1.id } });
        const finalAccount2 = await db.models.Account.findUnique({ where: { id: account2.id } });
        
        console.log('\nFinal balances:');
        console.log(`${finalAccount1.name}: $${finalAccount1.balance}`);
        console.log(`${finalAccount2.name}: $${finalAccount2.balance}`);
        
    } catch (error) {
        console.error('Transaction failed:', error.message);
    }
    
    // List all transactions
    const transactions = await db.models.Transaction.findMany();
    
    console.log('\nTransaction history:');
    transactions.forEach(tx => {
        console.log(`- $${tx.amount} from account ${tx.from_id} to account ${tx.to_id} at ${tx.created_at}`);
    });
    
    await db.close();
}

main().catch(err => {
    console.error('Error:', err.message || err);
    if (err.stack) {
        console.error('Stack:', err.stack);
    }
    process.exit(1);
});