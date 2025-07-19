#!/usr/bin/env python3
"""
AI Assistant Integration Example

This example shows how to integrate MCP with an AI assistant
for natural language database queries.

Requirements:
    pip install requests

Usage:
    1. Start MCP server: redi-orm mcp --db=sqlite://./assistant.db --transport=http --port=3000
    2. Run this script: python ai-assistant-integration.py
"""

import json
import requests
from typing import Dict, Any, List, Optional
from datetime import datetime

class MCPClient:
    """MCP Client for AI Assistant Integration"""
    
    def __init__(self, base_url: str = "http://localhost:3000", api_key: Optional[str] = None):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
        self.request_id = 1
        
        if api_key:
            self.session.headers['Authorization'] = f'Bearer {api_key}'
        
        # Initialize the connection
        self._initialize()
    
    def _initialize(self):
        """Initialize MCP connection"""
        result = self.call_method('initialize', {
            'protocolVersion': '2024-11-05',
            'capabilities': {},
            'clientInfo': {
                'name': 'ai-assistant-client',
                'version': '1.0.0'
            }
        })
        print(f"Connected to MCP server: {result['serverInfo']['name']} v{result['serverInfo']['version']}")
    
    def call_method(self, method: str, params: Dict[str, Any] = None) -> Any:
        """Call an MCP method"""
        payload = {
            'jsonrpc': '2.0',
            'method': method,
            'params': params or {},
            'id': self.request_id
        }
        self.request_id += 1
        
        response = self.session.post(self.base_url, json=payload)
        response.raise_for_status()
        
        result = response.json()
        if 'error' in result:
            raise Exception(f"MCP Error: {result['error']['message']}")
        
        return result.get('result')
    
    def call_tool(self, tool_name: str, arguments: Dict[str, Any]) -> Any:
        """Call an MCP tool"""
        result = self.call_method('tools/call', {
            'name': tool_name,
            'arguments': arguments
        })
        
        # Parse the tool result
        if result.get('isError'):
            raise Exception(f"Tool error: {result['content'][0]['text']}")
        
        return json.loads(result['content'][0]['text'])
    
    def query(self, sql: str, parameters: List[Any] = None) -> Dict[str, Any]:
        """Execute a SQL query"""
        return self.call_tool('query', {
            'sql': sql,
            'parameters': parameters or []
        })
    
    def get_tables(self) -> List[str]:
        """Get list of available tables"""
        result = self.call_tool('list_tables', {})
        return result['tables']
    
    def analyze_table(self, table: str, columns: List[str] = None) -> Dict[str, Any]:
        """Analyze a table's statistics"""
        args = {'table': table}
        if columns:
            args['columns'] = columns
        return self.call_tool('analyze_table', args)


class DatabaseAssistant:
    """AI Assistant that can answer questions about your database"""
    
    def __init__(self, mcp_client: MCPClient):
        self.mcp = mcp_client
        self.context = self._build_context()
    
    def _build_context(self) -> Dict[str, Any]:
        """Build context about the database"""
        tables = self.mcp.get_tables()
        context = {
            'tables': tables,
            'schemas': {}
        }
        
        # Get schema for each table
        for table in tables[:5]:  # Limit to first 5 tables for demo
            try:
                schema = self.mcp.call_tool('inspect_schema', {'table': table})
                context['schemas'][table] = schema
            except:
                pass
        
        return context
    
    def natural_language_query(self, question: str) -> str:
        """
        Convert natural language to SQL and execute.
        In a real implementation, this would use an LLM.
        """
        # Simple pattern matching for demo
        question_lower = question.lower()
        
        if "how many" in question_lower:
            # Count query
            for table in self.context['tables']:
                if table.lower() in question_lower:
                    result = self.mcp.query(f"SELECT COUNT(*) as count FROM {table}")
                    count = result['results'][0]['count']
                    return f"There are {count} records in the {table} table."
        
        elif "top" in question_lower and "by" in question_lower:
            # Top N query
            if "customers" in question_lower and "revenue" in question_lower:
                result = self.mcp.query("""
                    SELECT c.name, SUM(s.amount) as total_revenue
                    FROM customers c
                    JOIN sales s ON c.id = s.customer_id
                    WHERE s.status = 'completed'
                    GROUP BY c.id, c.name
                    ORDER BY total_revenue DESC
                    LIMIT 5
                """)
                
                response = "Top 5 customers by revenue:\n"
                for i, row in enumerate(result['results'], 1):
                    response += f"{i}. {row['name']}: ${row['total_revenue']:.2f}\n"
                return response
        
        elif "average" in question_lower or "avg" in question_lower:
            # Average calculation
            if "order" in question_lower or "sale" in question_lower:
                result = self.mcp.query(
                    "SELECT AVG(amount) as avg_amount FROM sales WHERE status = 'completed'"
                )
                avg = result['results'][0]['avg_amount']
                return f"The average order value is ${avg:.2f}"
        
        elif "trend" in question_lower or "over time" in question_lower:
            # Time series query
            result = self.mcp.query("""
                SELECT 
                    DATE(created_at) as date,
                    COUNT(*) as orders,
                    SUM(amount) as revenue
                FROM sales
                WHERE created_at >= date('now', '-7 days')
                GROUP BY DATE(created_at)
                ORDER BY date
            """)
            
            response = "Sales trend for the last 7 days:\n"
            for row in result['results']:
                response += f"{row['date']}: {row['orders']} orders, ${row['revenue']:.2f}\n"
            return response
        
        return "I couldn't understand that question. Try asking about counts, averages, or top records."
    
    def analyze_performance(self) -> str:
        """Analyze database performance and provide insights"""
        # Use batch queries for comprehensive analysis
        batch_result = self.mcp.call_tool('batch_query', {
            'queries': [
                {
                    'sql': "SELECT COUNT(*) as total_tables FROM sqlite_master WHERE type='table'",
                    'label': 'table_count'
                },
                {
                    'sql': """SELECT 
                              name, 
                              (SELECT COUNT(*) FROM pragma_table_info(name)) as column_count
                            FROM sqlite_master 
                            WHERE type='table' 
                            ORDER BY column_count DESC 
                            LIMIT 5""",
                    'label': 'largest_tables'
                },
                {
                    'sql': """SELECT 
                              COUNT(*) as total_records,
                              MIN(created_at) as oldest_record,
                              MAX(created_at) as newest_record
                            FROM sales""",
                    'label': 'data_overview'
                }
            ]
        })
        
        insights = ["Database Performance Analysis:\n"]
        
        for result in batch_result['results']:
            if result['success'] and result['results']:
                if result['label'] == 'table_count':
                    count = result['results'][0]['total_tables']
                    insights.append(f"• Total tables: {count}")
                
                elif result['label'] == 'largest_tables':
                    insights.append("• Largest tables by column count:")
                    for table in result['results'][:3]:
                        insights.append(f"  - {table['name']}: {table['column_count']} columns")
                
                elif result['label'] == 'data_overview':
                    data = result['results'][0]
                    insights.append(f"• Sales data: {data['total_records']} records")
                    insights.append(f"  Date range: {data['oldest_record']} to {data['newest_record']}")
        
        return "\n".join(insights)
    
    def suggest_optimizations(self, table: str) -> str:
        """Suggest optimizations for a table"""
        # Analyze the table
        analysis = self.mcp.analyze_table(table)
        
        suggestions = [f"Optimization suggestions for '{table}' table:\n"]
        
        # Check for high null counts
        for column, stats in analysis['statistics'].items():
            null_percentage = (stats['null_count'] / analysis['total_rows'] * 100) if analysis['total_rows'] > 0 else 0
            
            if null_percentage > 50:
                suggestions.append(f"• Column '{column}' has {null_percentage:.1f}% NULL values - consider removing or making optional")
            
            if stats['unique_count'] == analysis['total_rows'] and stats['data_type'] in ['string', 'integer']:
                suggestions.append(f"• Column '{column}' has all unique values - good candidate for primary key or index")
            
            if stats['unique_count'] < 10 and analysis['total_rows'] > 100:
                suggestions.append(f"• Column '{column}' has low cardinality ({stats['unique_count']} unique values) - consider creating an index")
        
        return "\n".join(suggestions)


def demo_conversation():
    """Demonstrate an AI assistant conversation"""
    print("=== AI ASSISTANT DATABASE DEMO ===\n")
    
    # Initialize MCP client
    mcp = MCPClient()
    assistant = DatabaseAssistant(mcp)
    
    # Simulate a conversation
    questions = [
        "How many customers do we have?",
        "What's the average order value?",
        "Show me the sales trend over time",
        "Who are the top customers by revenue?",
    ]
    
    for question in questions:
        print(f"Human: {question}")
        answer = assistant.natural_language_query(question)
        print(f"Assistant: {answer}\n")
    
    # Performance analysis
    print("Human: Can you analyze the database performance?")
    analysis = assistant.analyze_performance()
    print(f"Assistant: {analysis}\n")
    
    # Optimization suggestions
    print("Human: Any optimization suggestions for the sales table?")
    suggestions = assistant.suggest_optimizations('sales')
    print(f"Assistant: {suggestions}\n")


def advanced_ai_integration():
    """Advanced AI integration example with streaming and context"""
    print("\n=== ADVANCED AI INTEGRATION ===\n")
    
    mcp = MCPClient()
    
    # Example: Generate a natural language report from data
    print("Generating executive summary report...\n")
    
    # Gather data using batch queries
    report_data = mcp.call_tool('batch_query', {
        'queries': [
            {
                'sql': """SELECT 
                          COUNT(DISTINCT customer_id) as customers,
                          COUNT(*) as orders,
                          SUM(amount) as revenue,
                          AVG(amount) as avg_order
                        FROM sales 
                        WHERE status = 'completed'
                        AND created_at >= date('now', '-30 days')""",
                'label': 'monthly_summary'
            },
            {
                'sql': """SELECT 
                          p.category,
                          COUNT(s.id) as units_sold,
                          SUM(s.amount) as revenue
                        FROM sales s
                        JOIN products p ON s.product_id = p.id
                        WHERE s.status = 'completed'
                        AND s.created_at >= date('now', '-30 days')
                        GROUP BY p.category
                        ORDER BY revenue DESC
                        LIMIT 3""",
                'label': 'top_categories'
            },
            {
                'sql': """SELECT 
                          strftime('%w', created_at) as day_of_week,
                          COUNT(*) as order_count
                        FROM sales
                        WHERE created_at >= date('now', '-30 days')
                        GROUP BY day_of_week
                        ORDER BY order_count DESC
                        LIMIT 1""",
                'label': 'best_day'
            }
        ]
    })
    
    # Generate report from data
    print("EXECUTIVE SUMMARY - Last 30 Days\n")
    print("=" * 50 + "\n")
    
    for result in report_data['results']:
        if result['success'] and result['results']:
            if result['label'] == 'monthly_summary':
                data = result['results'][0]
                print(f"Key Metrics:")
                print(f"• Active Customers: {data['customers']:,}")
                print(f"• Total Orders: {data['orders']:,}")
                print(f"• Total Revenue: ${data['revenue']:,.2f}")
                print(f"• Average Order Value: ${data['avg_order']:.2f}\n")
            
            elif result['label'] == 'top_categories':
                print("Top Performing Categories:")
                for i, cat in enumerate(result['results'], 1):
                    print(f"{i}. {cat['category']}: ${cat['revenue']:,.2f} ({cat['units_sold']} units)")
                print()
            
            elif result['label'] == 'best_day':
                days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
                day_num = int(result['results'][0]['day_of_week'])
                count = result['results'][0]['order_count']
                print(f"Best Sales Day: {days[day_num]} (avg {count} orders)\n")
    
    # Example: Stream large dataset for ML training
    print("\nPreparing data for ML model training...")
    
    # Get sample of recent sales data
    stream_result = mcp.call_tool('stream_query', {
        'sql': """SELECT 
                    amount,
                    quantity,
                    discount,
                    strftime('%H', created_at) as hour,
                    strftime('%w', created_at) as day_of_week,
                    CASE WHEN status = 'completed' THEN 1 ELSE 0 END as completed
                  FROM sales
                  WHERE created_at >= date('now', '-90 days')""",
        'batch_size': 100
    })
    
    print(f"• Total training samples: {stream_result['total_rows']}")
    print(f"• Batch size: {stream_result['batch_size']}")
    print(f"• Number of batches: {stream_result['batch_count']}")
    print("\nFirst batch sample (for feature engineering):")
    
    if stream_result['batches']:
        sample = stream_result['batches'][0][0]
        print(f"  Features: {list(sample.keys())}")
        print(f"  Sample: {sample}")


if __name__ == "__main__":
    try:
        # Run the basic demo
        demo_conversation()
        
        # Run advanced integration
        advanced_ai_integration()
        
        print("\n=== Demo completed successfully! ===")
        
    except Exception as e:
        print(f"Error: {e}")
        print("\nMake sure the MCP server is running:")
        print("redi-orm mcp --db=sqlite://./assistant.db --transport=http --port=3000")