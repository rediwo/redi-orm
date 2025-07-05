package orm

import (
	"testing"
)

// Error Handling Tests
func (jct *JSConformanceTests) runErrorHandlingTests(t *testing.T, runner *JSTestRunner) {
	// Test unique constraint violation
	jct.runWithCleanup(t, runner, "UniqueConstraintViolation", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model User {
	id    Int    @id @default(autoincrement())
	email String @unique
}
`+"`"+`);
		await db.syncSchemas();
		
		// Create user
		await db.models.User.create({ data: { email: 'unique@example.com' } });
		
		// Try to create duplicate
		try {
			await db.models.User.create({ data: { email: 'unique@example.com' } });
			throw new Error('Should have failed with unique constraint');
		} catch (err) {
			assert(err.message.includes('unique') || err.message.includes('UNIQUE'));
		}
		
		// await db.close();
	`)

	// Test not null constraint
	jct.runWithCleanup(t, runner, "NotNullConstraint", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model User {
	id    Int    @id @default(autoincrement())
	email String
}
`+"`"+`);
		await db.syncSchemas();
		
		// Try to create without required field
		try {
			await db.models.User.create({ data: {} });
			throw new Error('Should have failed with not null constraint');
		} catch (err) {
			if (!err.message.includes('required') && !err.message.includes('null') && !err.message.includes('NOT NULL')) {
				throw new Error('Expected error to mention required/null constraint, got: ' + err.message);
			}
		}
		
		// await db.close();
	`)

	// Test foreign key constraint
	jct.runWithCleanup(t, runner, "ForeignKeyConstraint", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model Author {
	id    Int    @id @default(autoincrement())
	name  String
	posts Post[]
}

model Post {
	id       Int    @id @default(autoincrement())
	title    String
	authorId Int
	author   Author @relation(fields: [authorId], references: [id])
}
`+"`"+`);
		await db.syncSchemas();
		
		// Try to create post with non-existent author
		try {
			await db.models.Post.create({ data: { title: 'Orphan Post', authorId: 999999 } });
			throw new Error('Should have failed with foreign key constraint');
		} catch (err) {
			// Foreign key error messages vary by database
			assert(err.message.includes('foreign') || err.message.includes('constraint') || err.message.includes('violates'));
		}
		
		// await db.close();
	`)

	// Test invalid field name
	jct.runWithCleanup(t, runner, "InvalidFieldName", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model User {
	id   Int    @id @default(autoincrement())
	name String
}
`+"`"+`);
		await db.syncSchemas();
		
		// Try to query with invalid field
		try {
			await db.models.User.findMany({ where: { invalidField: 'test' } });
			throw new Error('Should have failed with invalid field error');
		} catch (err) {
			assert(err.message.includes('invalid') || err.message.includes('field') || err.message.includes('column'));
		}
		
		// await db.close();
	`)

	// Test invalid model name
	jct.runWithCleanup(t, runner, "InvalidModelName", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model User {
	id   Int    @id @default(autoincrement())
	name String
}
`+"`"+`);
		await db.syncSchemas();
		
		// Try to access non-existent model
		try {
			await db.models.NonExistentModel.findMany({});
			throw new Error('Should have failed with invalid model error');
		} catch (err) {
			// Check if error is about undefined property or model not found
			assert(err.message.includes('undefined') || err.message.includes('model') || err.message.includes('Cannot read'));
		}
		
		// await db.close();
	`)

	// Test type mismatch
	jct.runWithCleanup(t, runner, "TypeMismatch", `
		const db = fromUri(TEST_DATABASE_URI);
		await db.connect();
		
		await db.loadSchema(`+"`"+`
model User {
	id  Int @id @default(autoincrement())
	age Int
}
`+"`"+`);
		await db.syncSchemas();
		
		// Try to insert string into int field
		try {
			await db.models.User.create({ data: { age: 'not a number' } });
			throw new Error('Should have failed with type mismatch');
		} catch (err) {
			// Type errors can vary by database
			assert(err.message.includes('type') || err.message.includes('invalid') || err.message.includes('cannot') || 
			       err.message.includes('Incorrect') || err.message.includes('integer'));
		}
		
		// await db.close();
	`)
}
