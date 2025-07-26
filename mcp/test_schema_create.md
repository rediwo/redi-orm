# Schema.create 测试用例

## 测试 1: 简单模型（无依赖）

```json
{
  "model": "User",
  "fields": [
    {"name": "id", "type": "String", "primaryKey": true, "default": "cuid()"},
    {"name": "name", "type": "String"},
    {"name": "email", "type": "String", "unique": true},
    {"name": "createdAt", "type": "DateTime", "default": "now()"}
  ]
}
```

**期望结果**: 
- 创建 User schema 文件
- 在数据库中创建 users 表
- `tables_created: ["users"]`
- `pending_schemas: []`

## 测试 2: 有依赖的模型（正常顺序）

先创建 User，再创建 Post：

```json
{
  "model": "Post", 
  "fields": [
    {"name": "id", "type": "String", "primaryKey": true, "default": "cuid()"},
    {"name": "title", "type": "String"},
    {"name": "content", "type": "String"},
    {"name": "authorId", "type": "String"},
    {"name": "createdAt", "type": "DateTime", "default": "now()"}
  ],
  "relations": [
    {
      "name": "author",
      "type": "manyToOne", 
      "model": "User",
      "foreignKey": "authorId",
      "references": "id"
    }
  ]
}
```

**期望结果**:
- 创建 Post schema 文件  
- 在数据库中创建 posts 表（因为 users 表已存在）
- `tables_created: ["posts"]`
- `pending_schemas: []`

## 测试 3: 依赖缺失的模型

先创建 Comment（依赖 Post 和 User）：

```json
{
  "model": "Comment",
  "fields": [
    {"name": "id", "type": "String", "primaryKey": true, "default": "cuid()"},
    {"name": "content", "type": "String"},
    {"name": "postId", "type": "String"},
    {"name": "authorId", "type": "String"}
  ],
  "relations": [
    {
      "name": "post", 
      "type": "manyToOne",
      "model": "Post", 
      "foreignKey": "postId",
      "references": "id"
    },
    {
      "name": "author",
      "type": "manyToOne",
      "model": "User",
      "foreignKey": "authorId", 
      "references": "id"
    }
  ]
}
```

**期望结果**:
- 创建 Comment schema 文件
- 如果 Post 和 User 表存在，创建 comments 表
- 如果依赖表不存在，`pending_schemas: ["Comment"]`

## 测试 4: 循环依赖

创建两个相互依赖的模型：

**Author 模型**:
```json
{
  "model": "Author",
  "fields": [
    {"name": "id", "type": "String", "primaryKey": true, "default": "cuid()"},
    {"name": "name", "type": "String"},
    {"name": "favoritePostId", "type": "String", "optional": true}
  ],
  "relations": [
    {
      "name": "favoritePost",
      "type": "oneToOne",
      "model": "BlogPost", 
      "foreignKey": "favoritePostId",
      "references": "id"
    }
  ]
}
```

**BlogPost 模型**:
```json
{
  "model": "BlogPost",
  "fields": [
    {"name": "id", "type": "String", "primaryKey": true, "default": "cuid()"},
    {"name": "title", "type": "String"},
    {"name": "authorId", "type": "String"}
  ],
  "relations": [
    {
      "name": "author",
      "type": "manyToOne",
      "model": "Author",
      "foreignKey": "authorId", 
      "references": "id"
    }
  ]
}
```

**期望结果**:
- 对于 MongoDB: 正常创建所有表
- 对于 SQL 数据库: `has_circular_dependencies: true`，返回错误信息指导手动处理

## 测试命令

1. 启动 MCP 服务器: `./redi-mcp --db=sqlite://./test.db --port=8001 --read-only=false`
2. 使用 MCP 客户端发送 `schema.create` 请求
3. 检查返回的 JSON 结果
4. 验证数据库中的表是否正确创建

## 期望的功能提升

1. **智能依赖处理**: 自动检测并按正确顺序创建表
2. **清晰的状态反馈**: 详细报告哪些表已创建，哪些还在等待
3. **错误处理**: 提供有用的错误信息和解决建议
4. **循环依赖支持**: 对于支持的数据库，智能处理循环依赖