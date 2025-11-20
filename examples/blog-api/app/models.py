"""Pydantic models for request/response validation"""

from pydantic import BaseModel, EmailStr, Field
from datetime import datetime
from typing import Optional


# User models
class UserCreate(BaseModel):
    """Request model for creating a user"""
    username: str = Field(..., min_length=3, max_length=50)
    email: EmailStr
    full_name: Optional[str] = None
    bio: Optional[str] = None


class UserUpdate(BaseModel):
    """Request model for updating a user"""
    email: EmailStr
    full_name: Optional[str] = None
    bio: Optional[str] = None


class UserResponse(BaseModel):
    """Response model for user data"""
    user_id: int
    username: str
    email: str
    full_name: Optional[str]
    bio: Optional[str]
    created_at: datetime
    updated_at: datetime
    
    class Config:
        from_attributes = True


# Post models
class PostCreate(BaseModel):
    """Request model for creating a post"""
    title: str = Field(..., min_length=1, max_length=255)
    content: str = Field(..., min_length=1)
    published: bool = False


class PostUpdate(BaseModel):
    """Request model for updating a post"""
    title: str = Field(..., min_length=1, max_length=255)
    content: str = Field(..., min_length=1)
    published: bool


class PostResponse(BaseModel):
    """Response model for post data"""
    post_id: int
    title: str
    content: str
    author_id: int
    published: bool
    view_count: int
    created_at: datetime
    updated_at: datetime
    created_by: Optional[int]
    updated_by: Optional[int]
    
    class Config:
        from_attributes = True


class PostWithAuthor(BaseModel):
    """Response model for post with author information"""
    post_id: int
    title: str
    content: str
    author_id: int
    published: bool
    view_count: int
    created_at: datetime
    updated_at: datetime
    author_username: str
    author_full_name: Optional[str]
    
    class Config:
        from_attributes = True


# Comment models
class CommentCreate(BaseModel):
    """Request model for creating a comment"""
    post_id: int
    content: str = Field(..., min_length=1)


class CommentResponse(BaseModel):
    """Response model for comment data"""
    comment_id: int
    post_id: int
    author_id: int
    content: str
    created_at: datetime
    updated_at: datetime
    
    class Config:
        from_attributes = True


class CommentWithAuthor(BaseModel):
    """Response model for comment with author information"""
    comment_id: int
    post_id: int
    author_id: int
    content: str
    created_at: datetime
    updated_at: datetime
    author_username: str
    author_full_name: Optional[str]
    
    class Config:
        from_attributes = True


# Pagination models
class PaginationParams(BaseModel):
    """Query parameters for pagination"""
    limit: int = Field(default=20, ge=1, le=100)
    offset: int = Field(default=0, ge=0)


# Search models
class SearchParams(BaseModel):
    """Query parameters for search"""
    query: str = Field(..., min_length=1)
    limit: int = Field(default=20, ge=1, le=100)


# Count response
class CountResponse(BaseModel):
    """Response model for count queries"""
    total: int
