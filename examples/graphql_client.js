// Example: GraphQL Client Usage
// This example shows how to interact with the GraphQL API programmatically

const fetch = require('node-fetch');

// GraphQL endpoint (assuming server is running on port 4000)
const GRAPHQL_ENDPOINT = 'http://localhost:4000/graphql';

// Helper function to make GraphQL requests
async function graphqlRequest(query, variables = {}) {
    const response = await fetch(GRAPHQL_ENDPOINT, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({
            query,
            variables
        })
    });
    
    const result = await response.json();
    
    if (result.errors) {
        throw new Error(JSON.stringify(result.errors, null, 2));
    }
    
    return result.data;
}

async function main() {
    console.log('GraphQL Client Example');
    console.log('======================\n');
    
    try {
        // 1. Query all users
        console.log('1. Fetching all users...');
        const usersQuery = `
            query GetUsers {
                findManyUser(orderBy: { name: ASC }) {
                    id
                    name
                    email
                    role
                    _count {
                        posts
                        comments
                    }
                }
            }
        `;
        
        const users = await graphqlRequest(usersQuery);
        console.log('Users:', JSON.stringify(users.findManyUser, null, 2));
        
        // 2. Create a new user
        console.log('\n2. Creating a new user...');
        const createUserMutation = `
            mutation CreateUser($data: UserCreateInput!) {
                createUser(data: $data) {
                    id
                    name
                    email
                    role
                }
            }
        `;
        
        const newUser = await graphqlRequest(createUserMutation, {
            data: {
                name: 'Charlie Brown',
                email: 'charlie@example.com',
                role: 'editor'
            }
        });
        console.log('Created user:', newUser.createUser);
        
        // 3. Query posts with pagination and filtering
        console.log('\n3. Fetching posts with pagination...');
        const postsQuery = `
            query GetPosts($where: PostWhereInput, $limit: Int, $offset: Int) {
                findManyPost(
                    where: $where
                    limit: $limit
                    offset: $offset
                    orderBy: { createdAt: DESC }
                ) {
                    id
                    title
                    published
                    author {
                        name
                    }
                    _count {
                        comments
                    }
                }
                countPost(where: $where)
            }
        `;
        
        const postsResult = await graphqlRequest(postsQuery, {
            where: { published: { equals: true } },
            limit: 10,
            offset: 0
        });
        
        console.log(`Found ${postsResult.countPost} published posts`);
        console.log('First page:', JSON.stringify(postsResult.findManyPost, null, 2));
        
        // 4. Create a post with nested comment
        console.log('\n4. Creating a post with a comment...');
        const createPostMutation = `
            mutation CreatePostWithComment($data: PostCreateInput!) {
                createPost(data: $data) {
                    id
                    title
                    author {
                        name
                    }
                    comments {
                        content
                        author {
                            name
                        }
                    }
                }
            }
        `;
        
        // Note: Nested creates would require the GraphQL schema to support them
        // For now, we'll create the post and comment separately
        const createPostSimple = `
            mutation CreatePost($data: PostCreateInput!) {
                createPost(data: $data) {
                    id
                    title
                    author {
                        name
                    }
                }
            }
        `;
        
        const newPost = await graphqlRequest(createPostSimple, {
            data: {
                title: 'GraphQL is Amazing!',
                content: 'Let me tell you why GraphQL is so powerful...',
                published: true,
                authorId: newUser.createUser.id
            }
        });
        console.log('Created post:', newPost.createPost);
        
        // 5. Batch operations
        console.log('\n5. Performing batch operations...');
        const batchCreateMutation = `
            mutation CreateManyTags($data: [TagCreateInput!]!) {
                createManyTag(data: $data) {
                    count
                }
            }
        `;
        
        const batchResult = await graphqlRequest(batchCreateMutation, {
            data: [
                { name: 'GraphQL' },
                { name: 'API' },
                { name: 'Tutorial' }
            ]
        });
        console.log(`Created ${batchResult.createManyTag.count} tags`);
        
        // 6. Complex query with multiple operations
        console.log('\n6. Executing complex query...');
        const complexQuery = `
            query GetDashboardData {
                userCount: countUser
                postCount: countPost
                publishedPostCount: countPost(where: { published: { equals: true } })
                
                recentPosts: findManyPost(
                    limit: 5
                    orderBy: { createdAt: DESC }
                ) {
                    id
                    title
                    createdAt
                }
                
                topAuthors: findManyUser(
                    limit: 3
                    orderBy: { posts: { _count: DESC } }
                ) {
                    name
                    _count {
                        posts
                    }
                }
            }
        `;
        
        const dashboardData = await graphqlRequest(complexQuery);
        console.log('Dashboard data:', JSON.stringify(dashboardData, null, 2));
        
    } catch (error) {
        console.error('GraphQL Error:', error.message);
        console.log('\nMake sure the GraphQL server is running:');
        console.log('  redi-orm server --db=sqlite://./blog.db --port=4000');
    }
}

// Note: This example requires the 'node-fetch' package
// Install it with: npm install node-fetch@2

main().catch(console.error);