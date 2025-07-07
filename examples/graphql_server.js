// Example: GraphQL Server with RediORM
// This script demonstrates how to use the built-in GraphQL server

const { fromUri } = require('redi/orm');

async function main() {
    // Create database connection
    const db = fromUri('sqlite://./blog.db');
    await db.connect();
    
    // Define schema
    await db.loadSchema(`
        model User {
            id        Int      @id @default(autoincrement())
            email     String   @unique
            name      String
            role      String   @default("user")
            posts     Post[]
            comments  Comment[]
            createdAt DateTime @default(now())
            updatedAt DateTime @updatedAt
        }
        
        model Post {
            id         Int       @id @default(autoincrement())
            title      String
            content    String
            published  Boolean   @default(false)
            authorId   Int
            author     User      @relation(fields: [authorId], references: [id])
            comments   Comment[]
            tags       Tag[]
            createdAt  DateTime  @default(now())
            updatedAt  DateTime  @updatedAt
        }
        
        model Comment {
            id        Int      @id @default(autoincrement())
            content   String
            postId    Int
            post      Post     @relation(fields: [postId], references: [id])
            authorId  Int
            author    User     @relation(fields: [authorId], references: [id])
            createdAt DateTime @default(now())
        }
        
        model Tag {
            id    Int    @id @default(autoincrement())
            name  String @unique
            posts Post[]
        }
    `);
    
    // Sync schemas with database
    await db.syncSchemas();
    
    console.log('Database schema created successfully!');
    
    // Create some sample data
    const { User, Post, Tag } = db.models;
    
    // Create users
    const alice = await User.create({
        data: {
            email: 'alice@example.com',
            name: 'Alice Johnson',
            role: 'admin'
        }
    });
    
    const bob = await User.create({
        data: {
            email: 'bob@example.com',
            name: 'Bob Smith'
        }
    });
    
    // Create tags
    const techTag = await Tag.create({
        data: { name: 'Technology' }
    });
    
    const jsTag = await Tag.create({
        data: { name: 'JavaScript' }
    });
    
    // Create posts
    const post1 = await Post.create({
        data: {
            title: 'Getting Started with GraphQL',
            content: 'GraphQL is a powerful query language for APIs...',
            published: true,
            authorId: alice.id
        }
    });
    
    const post2 = await Post.create({
        data: {
            title: 'Building APIs with RediORM',
            content: 'RediORM makes it easy to create GraphQL APIs...',
            published: true,
            authorId: alice.id
        }
    });
    
    // Create comments
    await db.models.Comment.create({
        data: {
            content: 'Great article!',
            postId: post1.id,
            authorId: bob.id
        }
    });
    
    console.log('\nSample data created!');
    console.log('\nTo start the GraphQL server, run:');
    console.log('  redi-orm server --db=sqlite://./blog.db --port=4000 --playground=true');
    console.log('\nThen visit http://localhost:4000/graphql to explore the API');
    
    console.log('\nExample GraphQL queries:');
    console.log(`
# Get all posts with authors and comments
query GetPosts {
  findManyPost(
    where: { published: { equals: true } }
    orderBy: { createdAt: DESC }
  ) {
    id
    title
    content
    author {
      name
      email
    }
    comments {
      content
      author {
        name
      }
    }
    _count {
      comments
    }
  }
}

# Create a new post
mutation CreatePost {
  createPost(data: {
    title: "My New Post"
    content: "This is the content..."
    authorId: 1
    published: true
  }) {
    id
    title
    author {
      name
    }
  }
}

# Update a post
mutation UpdatePost {
  updatePost(
    where: { id: { equals: 1 } }
    data: { published: false }
  ) {
    id
    published
  }
}

# Complex filtering
query ComplexFilter {
  findManyPost(
    where: {
      AND: [
        { published: { equals: true } }
        { title: { contains: "GraphQL" } }
      ]
    }
  ) {
    title
    author {
      name
    }
  }
}

# Count posts by author
query CountPosts {
  countPost(where: { authorId: { equals: 1 } })
}
    `);
    
    await db.close();
}

main().catch(console.error);