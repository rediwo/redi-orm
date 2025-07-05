package orm

import (
	"testing"
)

// Transaction Tests
func (jct *JSConformanceTests) runTransactionTests(t *testing.T, runner *JSTestRunner) {
	// Test basic transaction
	jct.runWithCleanup(t, runner, "BasicTransaction", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Account {
	id      Int    @id @default(autoincrement())
	name    String
	balance Float
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create accounts
		const alice = await db.models.Account.create({
			data: { name: 'Alice', balance: 1000 }
		});
		const bob = await db.models.Account.create({
			data: { name: 'Bob', balance: 500 }
		});
		
		// Test successful transaction
		await db.transaction(async (tx) => {
			// Deduct from Alice
			await tx.models.Account.update({
				where: { id: alice.id },
				data: { balance: 900 }
			});
			
			// Add to Bob
			await tx.models.Account.update({
				where: { id: bob.id },
				data: { balance: 600 }
			});
		});
		
		// Verify balances
		const aliceAfter = await db.models.Account.findUnique({ where: { id: alice.id } });
		const bobAfter = await db.models.Account.findUnique({ where: { id: bob.id } });
		
		assert.strictEqual(aliceAfter.balance, 900);
		assert.strictEqual(bobAfter.balance, 600);
		
		// await db.close();
	`)

	// Test transaction rollback
	jct.runWithCleanup(t, runner, "TransactionRollback", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Account {
	id      Int    @id @default(autoincrement())
	name    String
	balance Float
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create account
		const account = await db.models.Account.create({
			data: { name: 'Test', balance: 1000 }
		});
		
		// Test failed transaction
		try {
			await db.transaction(async (tx) => {
				// Update balance
				await tx.models.Account.update({
					where: { id: account.id },
					data: { balance: 500 }
				});
				
				// Force error
				throw new Error('Transaction failed');
			});
		} catch (err) {
			// Expected error
		}
		
		// Verify balance unchanged
		const accountAfter = await db.models.Account.findUnique({ where: { id: account.id } });
		assert.strictEqual(accountAfter.balance, 1000);
		
		// await db.close();
	`)

	// Test nested transactions (savepoints)
	jct.runWithCleanup(t, runner, "NestedTransactions", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Counter {
	id    Int    @id @default(autoincrement())
	name  String
	value Int
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create test data
		const counter = await db.models.Counter.create({
			data: { name: 'Test', value: 0 }
		});
		
		// Test nested transactions
		try {
			await db.transaction(async (tx1) => {
				// Outer transaction
				await tx1.models.Counter.update({
					where: { id: counter.id },
					data: { value: 1 }
				});
				
				// Nested transaction
				await tx1.transaction(async (tx2) => {
					await tx2.models.Counter.update({
						where: { id: counter.id },
						data: { value: 2 }
					});
					
					// Force rollback of inner transaction
					throw new Error('Rollback inner');
				});
			});
		} catch (err) {
			// Expected error
		}
		
		// Verify value is rolled back to 0 (both transactions rolled back)
		const afterRollback = await db.models.Counter.findUnique({ where: { id: counter.id } });
		assert.strictEqual(afterRollback.value, 0);
		
		// await db.close();
	`)

	// Test transaction isolation
	if !jct.shouldSkip("TestTransactionIsolation") {
		jct.runWithCleanup(t, runner, "TransactionIsolation", `
			const db = fromUri(TEST_DATABASE_URI);
			await db.connect();
			
			await db.loadSchema(`+"`"+`
model Counter {
	id    Int    @id @default(autoincrement())
	name  String
	value Int
}
`+"`"+`);
			await db.syncSchemas();
			
			// Create test data
			const counter = await db.models.Counter.create({
				data: { name: 'Isolation Test', value: 0 }
			});
			
			// Start a transaction that doesn't commit immediately
			let transactionComplete = false;
			const transactionPromise = db.transaction(async (tx) => {
				// Update value inside transaction
				await tx.models.Counter.update({
					where: { id: counter.id },
					data: { value: 100 }
				});
				
				// Read inside transaction should see the update
				const insideTx = await tx.models.Counter.findUnique({ where: { id: counter.id } });
				assert.strictEqual(insideTx.value, 100);
				
				// Simulate some work
				await new Promise(resolve => setTimeout(resolve, 100));
				transactionComplete = true;
			});
			
			// Read outside transaction (should see original value)
			const outside = await db.models.Counter.findUnique({ where: { id: counter.id } });
			assert.strictEqual(outside.value, 0);
			assert.strictEqual(transactionComplete, false);
			
			// Wait for transaction to complete
			await transactionPromise;
			
			// Now should see the updated value
			const afterCommit = await db.models.Counter.findUnique({ where: { id: counter.id } });
			assert.strictEqual(afterCommit.value, 100);
			
			// await db.close();
		`)
	}

	// Test concurrent transactions
	if !jct.shouldSkip("TestTransactionConcurrentAccess") {
		jct.runWithCleanup(t, runner, "TransactionConcurrentAccess", `
			const db = fromUri(TEST_DATABASE_URI);
			await db.connect();
			
			await db.loadSchema(`+"`"+`
model Balance {
	id     Int    @id @default(autoincrement())
	userId String @unique
	amount Float
}
`+"`"+`);
			await db.syncSchemas();
			
			// Create test data
			await db.models.Balance.create({
				data: { userId: 'user1', amount: 1000 }
			});
			
			// Run two concurrent transactions
			const tx1Promise = db.transaction(async (tx) => {
				const balance = await tx.models.Balance.findUnique({
					where: { userId: 'user1' }
				});
				
				await new Promise(resolve => setTimeout(resolve, 50));
				
				await tx.models.Balance.update({
					where: { userId: 'user1' },
					data: { amount: balance.amount + 100 }
				});
			});
			
			const tx2Promise = db.transaction(async (tx) => {
				const balance = await tx.models.Balance.findUnique({
					where: { userId: 'user1' }
				});
				
				await new Promise(resolve => setTimeout(resolve, 50));
				
				await tx.models.Balance.update({
					where: { userId: 'user1' },
					data: { amount: balance.amount + 200 }
				});
			});
			
			// Wait for both transactions
			await Promise.all([tx1Promise, tx2Promise]);
			
			// Check final balance
			const finalBalance = await db.models.Balance.findUnique({
				where: { userId: 'user1' }
			});
			
			// Due to isolation, one transaction might overwrite the other
			// The exact result depends on the database's isolation level
			assert(finalBalance.amount >= 1100); // At least one transaction succeeded
			
			// await db.close();
		`)
	}
}
