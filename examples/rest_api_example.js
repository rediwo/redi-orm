// REST API Example
// This example demonstrates how to use the REST API with RediORM

const API_BASE = 'http://localhost:8080/api';

// Helper function to make API calls
async function apiCall(method, path, body = null) {
    const options = {
        method: method,
        headers: {
            'Content-Type': 'application/json',
            'X-Connection-Name': 'default' // Use default connection
        }
    };
    
    if (body) {
        options.body = JSON.stringify(body);
    }
    
    const response = await fetch(`${API_BASE}${path}`, options);
    const data = await response.json();
    
    if (!data.success) {
        throw new Error(`API Error: ${data.error.message}`);
    }
    
    return data;
}

// Example usage
async function main() {
    try {
        console.log('=== REST API Example ===\n');
        
        // 1. Create a user
        console.log('1. Creating a user...');
        const createResponse = await apiCall('POST', '/users', {
            data: {
                name: 'John Doe',
                email: 'john@example.com',
                age: 30
            }
        });
        console.log('Created user:', createResponse.data);
        const userId = createResponse.data.id;
        
        // 2. Get all users
        console.log('\n2. Getting all users...');
        const usersResponse = await apiCall('GET', '/users');
        console.log('All users:', usersResponse.data);
        
        // 3. Get a specific user
        console.log('\n3. Getting user by ID...');
        const userResponse = await apiCall('GET', `/users/${userId}`);
        console.log('User details:', userResponse.data);
        
        // 4. Update the user
        console.log('\n4. Updating user...');
        const updateResponse = await apiCall('PUT', `/users/${userId}`, {
            data: {
                age: 31
            }
        });
        console.log('Updated user:', updateResponse.data);
        
        // 5. Query with filters
        console.log('\n5. Querying with filters...');
        const filteredResponse = await apiCall('GET', '/users?filter[age][gt]=25&sort=-name&limit=10');
        console.log('Filtered users:', filteredResponse.data);
        
        // 6. Create a post for the user
        console.log('\n6. Creating a post...');
        const postResponse = await apiCall('POST', '/posts', {
            data: {
                title: 'My First Post',
                content: 'This is the content of my first post',
                published: true,
                authorId: userId
            }
        });
        console.log('Created post:', postResponse.data);
        
        // 7. Get user with posts (include relation)
        console.log('\n7. Getting user with posts...');
        const userWithPostsResponse = await apiCall('GET', `/users/${userId}?include=posts`);
        console.log('User with posts:', JSON.stringify(userWithPostsResponse.data, null, 2));
        
        // 8. Batch create
        console.log('\n8. Batch creating users...');
        const batchResponse = await apiCall('POST', '/users/batch', {
            data: [
                { name: 'Alice', email: 'alice@example.com', age: 25 },
                { name: 'Bob', email: 'bob@example.com', age: 35 },
                { name: 'Charlie', email: 'charlie@example.com', age: 28 }
            ]
        });
        console.log('Batch create result:', batchResponse.data);
        
        // 9. Complex query with pagination
        console.log('\n9. Complex query with pagination...');
        const complexResponse = await apiCall('GET', '/users?page=1&limit=5&sort=name&select=id,name,email');
        console.log('Paginated response:', {
            data: complexResponse.data,
            pagination: complexResponse.pagination,
            meta: complexResponse.meta
        });
        
        // 10. Delete a user
        console.log('\n10. Deleting user...');
        const deleteResponse = await apiCall('DELETE', `/users/${userId}`);
        console.log('Delete result:', deleteResponse.data);
        
    } catch (error) {
        console.error('Error:', error.message);
    }
}

// Note: This example requires the REST server to be running
// Start it with: redi-orm rest-server --db=sqlite://./rest-example.db --schema=./schema.prisma
console.log('Make sure the REST server is running:');
console.log('redi-orm rest-server --db=sqlite://./rest-example.db --schema=./schema.prisma\n');

// Uncomment to run the example
// main();