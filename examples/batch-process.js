// Example: Batch processing with timeout
const { fromUri } = require('redi/orm');

async function main() {
    const db = fromUri('sqlite://./batch.db');
    await db.connect();
    
    await db.loadSchema(`
        model Record {
            id        Int      @id @default(autoincrement())
            data      String
            processed Boolean  @default(false)
            createdAt DateTime @default(now())
        }
    `);
    
    await db.syncSchemas();
    
    // Create some test records
    console.log('Creating test records...');
    const records = [];
    for (let i = 1; i <= 20; i++) {
        const record = await db.models.Record.create({
            data: { data: `Record ${i}` }
        });
        records.push(record);
    }
    console.log(`Created ${records.length} records`);
    
    // Process records in batches
    console.log('\nProcessing records in batches...');
    let processedCount = 0;
    const batchSize = 5;
    
    while (true) {
        // Get unprocessed records
        const batch = await db.models.Record.findMany({
            where: { processed: false },
            take: batchSize
        });
        
        if (batch.length === 0) {
            console.log('No more records to process');
            break;
        }
        
        console.log(`Processing batch of ${batch.length} records...`);
        
        // Process each record
        for (const record of batch) {
            // Simulate processing time
            await new Promise(resolve => setTimeout(resolve, 500));
            
            await db.models.Record.update({
                where: { id: record.id },
                data: { processed: true }
            });
            
            processedCount++;
            console.log(`  Processed: ${record.data}`);
        }
        
        console.log(`Batch complete. Total processed: ${processedCount}`);
    }
    
    await db.close();
    console.log('\nBatch processing completed!');
}

// Run with: ./redi-orm run --timeout=30000 examples/batch-process.js
main().catch(err => {
    console.error('Error:', err.message || err);
    process.exit(1);
});