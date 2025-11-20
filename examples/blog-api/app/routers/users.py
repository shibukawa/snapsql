"""User API endpoints"""

from datetime import datetime
from fastapi import APIRouter, Depends, HTTPException
from typing import List
import asyncpg

from app.database import get_db_connection
from app.models import UserCreate, UserUpdate, UserResponse

# Import generated query functions (raises ImportError if SnapSQL code is missing)
from dataaccess import (
    user_get,
    user_list,
    NotFoundError as DataAccessNotFoundError,
)

UserGetNotFoundError = DataAccessNotFoundError

router = APIRouter()


@router.post("/", response_model=UserResponse, status_code=201)
async def create_user(
    user: UserCreate,
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """Create a new user"""
    try:
        row = await conn.fetchrow(
            """
            INSERT INTO users (username, email, full_name, bio, created_at, updated_at)
            VALUES ($1, $2, $3, $4, $5, $6)
            RETURNING user_id, username, email, full_name, bio, created_at, updated_at
            """,
            user.username,
            user.email,
            user.full_name,
            user.bio,
            datetime.utcnow(),
            datetime.utcnow(),
        )

        if row is None:
            raise HTTPException(status_code=500, detail="Failed to insert user")

        return UserResponse(**dict(row))
    except asyncpg.UniqueViolationError:
        raise HTTPException(status_code=409, detail="Username or email already exists")
    except Exception as e:
        raise HTTPException(status_code=400, detail=str(e))


@router.get("/{user_id}", response_model=UserResponse)
async def get_user(
    user_id: int,
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """Get a user by ID"""
    try:
        result = await user_get(conn, user_id=user_id)
        return UserResponse(**result.to_dict())
    except UserGetNotFoundError:
        raise HTTPException(status_code=404, detail="User not found")
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/", response_model=List[UserResponse])
async def list_users(
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """List all users"""
    try:
        results: List[UserResponse] = []
        async for item in user_list(conn):
            results.append(UserResponse(**item.to_dict()))
        return results
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.put("/{user_id}", response_model=UserResponse)
async def update_user(
    user_id: int,
    user: UserUpdate,
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """Update a user"""
    try:
        # TODO: Use generated update_user function
        # result = await update_user(
        #     conn,
        #     user_id=user_id,
        #     email=user.email,
        #     full_name=user.full_name,
        #     bio=user.bio
        # )
        # if not result:
        #     raise HTTPException(status_code=404, detail="User not found")
        # return result
        
        raise HTTPException(
            status_code=501,
            detail="Not implemented - run SnapSQL code generation first"
        )
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.delete("/{user_id}", status_code=204)
async def delete_user(
    user_id: int,
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """Delete a user"""
    try:
        # TODO: Use generated delete_user function
        # await delete_user(conn, user_id=user_id)
        # return None
        
        raise HTTPException(
            status_code=501,
            detail="Not implemented - run SnapSQL code generation first"
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
