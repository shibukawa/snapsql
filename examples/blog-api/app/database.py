"""Database connection management"""

import asyncpg
from contextlib import asynccontextmanager
from typing import AsyncGenerator
import os

# Database configuration
DATABASE_URL = os.getenv(
    "DATABASE_URL",
    "postgresql://bloguser:blogpass@localhost:5432/blogdb"
)


class Database:
    """Database connection pool manager"""
    
    def __init__(self):
        self.pool: asyncpg.Pool | None = None
    
    async def connect(self):
        """Create database connection pool"""
        if self.pool is None:
            self.pool = await asyncpg.create_pool(
                DATABASE_URL,
                min_size=2,
                max_size=10,
                command_timeout=60
            )
    
    async def disconnect(self):
        """Close database connection pool"""
        if self.pool is not None:
            await self.pool.close()
            self.pool = None
    
    @asynccontextmanager
    async def connection(self) -> AsyncGenerator[asyncpg.Connection, None]:
        """Get a database connection from the pool"""
        if self.pool is None:
            await self.connect()
        
        async with self.pool.acquire() as conn:
            yield conn


# Global database instance
db = Database()


async def get_db_connection() -> AsyncGenerator[asyncpg.Connection, None]:
    """Dependency for getting database connection in FastAPI routes"""
    async with db.connection() as conn:
        yield conn
