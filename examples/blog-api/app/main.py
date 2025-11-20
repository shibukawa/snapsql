"""FastAPI Blog API Application"""

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from contextlib import asynccontextmanager
import sys
import os

# Add parent directory to path to import generated code
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from app.database import db
from app.routers import users, posts, comments


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Lifecycle manager for the application"""
    # Startup
    await db.connect()
    print("✓ Database connected")
    
    yield
    
    # Shutdown
    await db.disconnect()
    print("✓ Database disconnected")


# Create FastAPI application
app = FastAPI(
    title="Blog API",
    description="A sample blog API built with FastAPI and SnapSQL",
    version="1.0.0",
    lifespan=lifespan
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # In production, specify actual origins
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(users.router, prefix="/users", tags=["users"])
app.include_router(posts.router, prefix="/posts", tags=["posts"])
app.include_router(comments.router, prefix="/comments", tags=["comments"])


@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "message": "Welcome to Blog API",
        "version": "1.0.0",
        "docs": "/docs",
        "endpoints": {
            "users": "/users",
            "posts": "/posts",
            "comments": "/comments"
        }
    }


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {"status": "healthy"}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
