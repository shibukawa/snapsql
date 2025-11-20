"""Post API endpoints"""

from fastapi import APIRouter, Depends, HTTPException, Query
from typing import Any, Dict, List
import asyncpg

from app.database import get_db_connection
from app.models import PostCreate, PostUpdate, PostResponse, PostWithAuthor, PaginationParams

from dataaccess import (
    post_create,
    post_get,
    post_list,
    ValidationError as DataAccessValidationError,
    NotFoundError as DataAccessNotFoundError,
)

PostCreateValidationError = DataAccessValidationError
PostGetValidationError = DataAccessValidationError
PostListValidationError = DataAccessValidationError
PostGetNotFoundError = DataAccessNotFoundError

router = APIRouter()


@router.post("/", response_model=PostResponse, status_code=201)
async def create_post(
    post: PostCreate,
    author_id: int = Query(..., description="Author user ID"),
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """Create a new blog post"""
    try:
        result = await post_create(
            conn,
            title=post.title,
            content=post.content,
            author_id=author_id,
            published=post.published,
            created_by=author_id,
        )
        return PostResponse(**result.to_dict())
    except PostCreateValidationError as exc:
        raise HTTPException(status_code=400, detail=exc.message) from exc
    except asyncpg.UniqueViolationError as exc:
        raise HTTPException(status_code=409, detail="Post already exists") from exc
    except Exception as exc:  # pragma: no cover - unexpected failures bubble up as 500
        raise HTTPException(status_code=500, detail=str(exc)) from exc


@router.get("/{post_id}", response_model=PostWithAuthor)
async def get_post(
    post_id: int,
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """Get a post by ID with author information"""
    try:
        result = await post_get(conn, post_id=post_id)
        return build_post_with_author(result.to_dict())
    except PostGetNotFoundError as exc:
        raise HTTPException(status_code=404, detail=exc.message) from exc
    except PostGetValidationError as exc:
        raise HTTPException(status_code=400, detail=exc.message) from exc
    except Exception as exc:  # pragma: no cover - unexpected failures bubble up as 500
        raise HTTPException(status_code=500, detail=str(exc)) from exc


@router.get("/", response_model=List[PostWithAuthor])
async def list_posts(
    limit: int = Query(20, ge=1, le=100),
    offset: int = Query(0, ge=0),
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """List published posts with pagination"""
    try:
        results: List[PostWithAuthor] = []
        async for record in post_list(conn, limit=limit, offset=offset):
            results.append(build_post_with_author(record.to_dict()))
        return results
    except PostListValidationError as exc:
        raise HTTPException(status_code=400, detail=exc.message) from exc
    except Exception as exc:  # pragma: no cover - unexpected failures bubble up as 500
        raise HTTPException(status_code=500, detail=str(exc)) from exc


@router.put("/{post_id}", response_model=PostResponse)
async def update_post(
    post_id: int,
    post: PostUpdate,
    updated_by: int = Query(..., description="User ID making the update"),
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """Update a blog post"""
    try:
        # TODO: Use generated update_post function
        # result = await update_post(
        #     conn,
        #     post_id=post_id,
        #     title=post.title,
        #     content=post.content,
        #     published=post.published,
        #     updated_by=updated_by
        # )
        # if not result:
        #     raise HTTPException(status_code=404, detail="Post not found")
        # return result
        
        raise HTTPException(
            status_code=501,
            detail="Not implemented - run SnapSQL code generation first"
        )
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.delete("/{post_id}", status_code=204)
async def delete_post(
    post_id: int,
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """Delete a blog post"""
    try:
        # TODO: Use generated delete_post function
        # await delete_post(conn, post_id=post_id)
        # return None
        
        raise HTTPException(
            status_code=501,
            detail="Not implemented - run SnapSQL code generation first"
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


def build_post_with_author(payload: Dict[str, Any]) -> PostWithAuthor:
    """Convert SnapSQL dataclass dict (with nested author) to PostWithAuthor."""
    author_entries = payload.pop("author", []) or []
    author = author_entries[0] if author_entries else {}
    return PostWithAuthor(
        **payload,
        author_username=author.get("username", ""),
        author_full_name=author.get("full_name"),
    )
