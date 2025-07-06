// MongoDB Example for RediORM
// This demonstrates how to use RediORM with MongoDB

const { fromUri } = require('redi/orm');

async function main() {
    // Connect to MongoDB
    const db = fromUri('mongodb://localhost:27017/rediorm_example');
    await db.connect();
    
    try {
        // Load schema - MongoDB is schemaless but we can still define structure for validation
        await db.loadSchema(`
            model User {
                id       String @id @default(auto()) @map("_id") @db.ObjectId
                email    String @unique
                name     String
                age      Int?
                profile  Json?   // Nested document
                tags     String[]
                createdAt DateTime @default(now())
            }
            
            model Post {
                id        String   @id @default(auto()) @map("_id") @db.ObjectId
                title     String
                content   String
                published Boolean  @default(false)
                authorId  String   @db.ObjectId
                author    User     @relation(fields: [authorId], references: [id])
                tags      String[]
                metadata  Json?
                createdAt DateTime @default(now())
                updatedAt DateTime @updatedAt
            }
        `);
        
        // Sync schemas (creates collections and indexes)
        await db.syncSchemas();
        
        // Create a user with nested document
        const user = await db.models.User.create({
            data: {
                email: 'alice@example.com',
                name: 'Alice',
                age: 30,
                profile: {
                    bio: 'Software developer',
                    location: 'San Francisco',
                    social: {
                        twitter: '@alice',
                        github: 'alice-dev'
                    }
                },
                tags: ['developer', 'mongodb', 'nodejs']
            }
        });
        
        console.log('Created user:', user);
        
        // Create posts
        await db.models.Post.create({
            data: {
                title: 'Getting Started with MongoDB',
                content: 'MongoDB is a great NoSQL database...',
                authorId: user.id,
                tags: ['mongodb', 'tutorial', 'database'],
                metadata: {
                    readTime: '5 min',
                    difficulty: 'beginner'
                }
            }
        });
        
        await db.models.Post.create({
            data: {
                title: 'Advanced MongoDB Queries',
                content: 'Let\'s explore aggregation pipelines...',
                authorId: user.id,
                published: true,
                tags: ['mongodb', 'advanced', 'aggregation'],
                metadata: {
                    readTime: '10 min',
                    difficulty: 'advanced'
                }
            }
        });
        
        // Query with nested field conditions
        const sfUsers = await db.models.User.findMany({
            where: {
                'profile.location': 'San Francisco'
            }
        });
        console.log('Users in SF:', sfUsers);
        
        // Query with array contains
        const mongodbPosts = await db.models.Post.findMany({
            where: {
                tags: { in: ['mongodb'] }
            },
            include: {
                author: true
            }
        });
        console.log('MongoDB posts:', mongodbPosts);
        
        // Update nested fields
        await db.models.User.update({
            where: { id: user.id },
            data: {
                'profile.bio': 'Senior Software Developer',
                age: 31
            }
        });
        
        // Aggregation example (counts by tag)
        const tagCounts = await db.queryRaw(`{
            "aggregate": "posts",
            "pipeline": [
                { "$unwind": "$tags" },
                { "$group": {
                    "_id": "$tags",
                    "count": { "$sum": 1 }
                }},
                { "$sort": { "count": -1 } }
            ]
        }`);
        console.log('Tag counts:', tagCounts);
        
        // Transaction example
        await db.transaction(async (tx) => {
            // Create a new user
            const newUser = await tx.models.User.create({
                data: {
                    email: 'bob@example.com',
                    name: 'Bob',
                    tags: ['writer']
                }
            });
            
            // Create a post for the new user
            await tx.models.Post.create({
                data: {
                    title: 'My First Post',
                    content: 'Hello MongoDB!',
                    authorId: newUser.id,
                    tags: ['introduction']
                }
            });
            
            console.log('Transaction completed successfully');
        });
        
        // Complex query with multiple conditions
        const publishedTutorials = await db.models.Post.findMany({
            where: {
                AND: [
                    { published: true },
                    { tags: { contains: 'tutorial' } },
                    { 'metadata.difficulty': { in: ['beginner', 'intermediate'] } }
                ]
            },
            orderBy: {
                createdAt: 'desc'
            },
            include: {
                author: {
                    select: {
                        name: true,
                        email: true
                    }
                }
            }
        });
        console.log('Published tutorials:', publishedTutorials);
        
        // Count documents
        const userCount = await db.models.User.count();
        const publishedCount = await db.models.Post.count({
            where: { published: true }
        });
        
        console.log(`Total users: ${userCount}, Published posts: ${publishedCount}`);
        
    } finally {
        // Close connection
        await db.close();
    }
}

// Run the example
main().catch(console.error);