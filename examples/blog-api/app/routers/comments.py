"""Comment API endpoints"""

from fastapi import APIRouter, Depends, HTTPException, Query
from typing import Any, Dict, List
import asyncpg

from app.database import get_db_connection
from app.models import CommentCreate, CommentResponse, CommentWithAuthor

from dataaccess import (
    comment_create,
    comment_list_by_post,
    ValidationError as DataAccessValidationError,
)

CommentCreateValidationError = DataAccessValidationError
CommentListValidationError = DataAccessValidationError

router = APIRouter()


@router.post("/", response_model=CommentResponse, status_code=201)
async def create_comment(
    comment: CommentCreate,
    author_id: int = Query(..., description="Author user ID"),
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """Create a new comment on a post"""
    try:
        result = await comment_create(
            conn,
            post_id=comment.post_id,
            author_id=author_id,
            content=comment.content,
        )
        return CommentResponse(**result.to_dict())
    except CommentCreateValidationError as exc:
        raise HTTPException(status_code=400, detail=exc.message) from exc
    except Exception as exc:  # pragma: no cover - unexpected failures bubble up as 500
        raise HTTPException(status_code=500, detail=str(exc)) from exc


@router.get("/post/{post_id}", response_model=List[CommentWithAuthor])
async def list_comments_by_post(
    post_id: int,
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """Get all comments for a specific post"""
    try:
        results: List[CommentWithAuthor] = []
        async for record in comment_list_by_post(conn, post_id=post_id):
            results.append(build_comment_with_author(record.to_dict()))
        return results
    except CommentListValidationError as exc:
        raise HTTPException(status_code=400, detail=exc.message) from exc
    except Exception as exc:  # pragma: no cover - unexpected failures bubble up as 500
        raise HTTPException(status_code=500, detail=str(exc)) from exc


@router.delete("/{comment_id}", status_code=204)
async def delete_comment(
    comment_id: int,
    conn: asyncpg.Connection = Depends(get_db_connection)
):
    """Delete a comment"""
    try:
        # TODO: Use generated delete_comment function
        # await delete_comment(conn, comment_id=comment_id)
        # return None
        
        raise HTTPException(
            status_code=501,
            detail="Not implemented - run SnapSQL code generation first"
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


def build_comment_with_author(payload: Dict[str, Any]) -> CommentWithAuthor:
    """Convert SnapSQL dataclass dict (with nested author) to CommentWithAuthor."""
    author_entries = payload.pop("author", []) or []
    author = author_entries[0] if author_entries else {}
    return CommentWithAuthor(
        **payload,
        author_username=author.get("username", ""),
        author_full_name=author.get("full_name"),
    )
