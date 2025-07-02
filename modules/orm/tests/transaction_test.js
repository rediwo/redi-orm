// Transaction test
const { fromUri } = require('redi/orm');
const { assert, strictEqual, fail } = require('./assert');

async function setupDatabase() {
    const db = fromUri('sqlite://:memory:');
    await db.connect();
    
    const schema = `
model Account {
  id      Int    @id @default(autoincrement())
  name    String
  balance Float  @default(0)
}

model Transaction {
  id        Int      @id @default(autoincrement())
  fromId    Int
  toId      Int
  amount    Float
  createdAt DateTime @default(now())
}
`;
    
    await db.loadSchema(schema);
    await db.syncSchemas();
    return db;
}

async function createTestAccounts(db) {
    const alice = await db.models.Account.create({
        data: { name: 'Alice', balance: 1000 }
    });
    
    const bob = await db.models.Account.create({
        data: { name: 'Bob', balance: 500 }
    });
    
    return { alice, bob };
}

async function testSuccessfulTransaction(db) {
    console.log('Testing successful transaction...');
    
    try {
        const { alice, bob } = await createTestAccounts(db);
        const transferAmount = 100;
        
        await db.transaction(async (tx) => {
        // Deduct from Alice
        await tx.models.Account.update({
            where: { id: alice.id },
            data: { balance: alice.balance - transferAmount }
        });
        
        // Add to Bob
        await tx.models.Account.update({
            where: { id: bob.id },
            data: { balance: bob.balance + transferAmount }
        });
        
        // Record transaction
        await tx.models.Transaction.create({
            data: {
                fromId: alice.id,
                toId: bob.id,
                amount: transferAmount
            }
        });
    });
    
        // Verify balances
        const aliceAfter = await db.models.Account.findUnique({ where: { id: alice.id } });
        const bobAfter = await db.models.Account.findUnique({ where: { id: bob.id } });
        
        strictEqual(aliceAfter.balance, 900, 'Alice should have 900');
        strictEqual(bobAfter.balance, 600, 'Bob should have 600');
        
        console.log('  ✓ Transaction completed successfully');
        console.log('  ✓ Balances updated correctly');
    } catch (error) {
        console.error('  ❌ Error in testSuccessfulTransaction:', error.message || error);
        if (error.stack) {
            console.error('  Stack:', error.stack);
        }
        throw error;
    }
}

async function testFailedTransaction(db) {
    console.log('\nTesting failed transaction (rollback)...');
    
    const { alice, bob } = await createTestAccounts(db);
    const initialAliceBalance = alice.balance;
    const initialBobBalance = bob.balance;
    
    try {
        await db.transaction(async (tx) => {
            // Deduct from Alice
            await tx.models.Account.update({
                where: { id: alice.id },
                data: { balance: alice.balance - 2000 } // More than she has
            });
            
            // Check balance
            const aliceInTx = await tx.models.Account.findUnique({ where: { id: alice.id } });
            
            if (aliceInTx.balance < 0) {
                throw new Error('Insufficient funds');
            }
            
            // This should not execute
            await tx.models.Account.update({
                where: { id: bob.id },
                data: { balance: bob.balance + 2000 }
            });
        });
        
        fail('Transaction should have failed');
        
    } catch (error) {
        // The error message might be wrapped, so check if it contains the expected message
        const errorMessage = error.message || error.toString();
        if (errorMessage.includes('Insufficient funds')) {
            console.log('  ✓ Transaction failed as expected');
        } else {
            throw new Error(`Expected error to contain 'Insufficient funds', got: ${errorMessage}`);
        }
    }
    
    // Verify balances unchanged
    const aliceAfter = await db.models.Account.findUnique({ where: { id: alice.id } });
    const bobAfter = await db.models.Account.findUnique({ where: { id: bob.id } });
    
    strictEqual(aliceAfter.balance, initialAliceBalance, 'Alice balance should be unchanged');
    strictEqual(bobAfter.balance, initialBobBalance, 'Bob balance should be unchanged');
    
    console.log('  ✓ Rollback successful - balances unchanged');
}

async function testNestedOperations(db) {
    console.log('\nTesting transaction with multiple operations...');
    
    const accounts = [];
    
    // Create multiple accounts in a transaction
    await db.transaction(async (tx) => {
        for (let i = 0; i < 5; i++) {
            const account = await tx.models.Account.create({
                data: {
                    name: `Account ${i + 1}`,
                    balance: 100 * (i + 1)
                }
            });
            accounts.push(account);
        }
        
        // Create transactions between accounts
        for (let i = 0; i < accounts.length - 1; i++) {
            await tx.models.Transaction.create({
                data: {
                    fromId: accounts[i].id,
                    toId: accounts[i + 1].id,
                    amount: 10
                }
            });
        }
    });
    
    // Verify all accounts were created
    const accountCount = await db.models.Account.count();
    assert(accountCount >= 5, 'Should have created at least 5 accounts');
    
    // Verify transactions were created
    const transactionCount = await db.models.Transaction.count();
    assert(transactionCount >= 4, 'Should have created at least 4 transactions');
    
    console.log('  ✓ Multiple operations in transaction completed');
    console.log('  ✓ All data persisted correctly');
}

async function runTests() {
    console.log('=== Transaction Test Suite ===\n');
    
    let db;
    try {
        db = await setupDatabase();
        console.log('✓ Database setup complete\n');
        
        // Verify models exist
        console.log('Testing transaction models availability...');
        assert(typeof db.models === 'object', 'db.models should exist');
        assert(typeof db.models.Account === 'object', 'Account model should exist');
        assert(typeof db.models.Transaction === 'object', 'Transaction model should exist');
        console.log('  ✓ Models are available via db.models');
        
        // Verify methods exist
        const methods = ['create', 'findUnique', 'update', 'delete'];
        for (const method of methods) {
            assert(typeof db.models.Account[method] === 'function', `Account.${method} should exist`);
        }
        console.log('  ✓ All required methods exist');
        
        // Verify transaction method exists
        assert(typeof db.transaction === 'function', 'db.transaction should exist');
        console.log('  ✓ Transaction method exists\n');
        
        // Run transaction tests
        await testSuccessfulTransaction(db);
        await testFailedTransaction(db);
        await testNestedOperations(db);
        
        console.log('\n✅ All transaction tests passed!');
        
    } catch (error) {
        console.error('\n❌ Test failed:', error.message || error);
        if (error.stack) {
            console.error(error.stack);
        }
        process.exit(1);
    } finally {
        if (db) {
            await db.close();
        }
    }
}

runTests();