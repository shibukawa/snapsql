"""API integration tests for the Blog API."""

from datetime import datetime
from typing import AsyncIterator

import pytest
from fastapi.testclient import TestClient

from app.main import app
from app.database import get_db_connection
from app.routers import comments as comments_router
from app.routers import posts as posts_router

from dataaccess.comment_create import CommentCreateResult
from dataaccess.comment_list_by_post import (
    CommentListByPostResult,
    CommentListByPostResultAuthor,
)
from dataaccess.post_create import PostCreateResult
from dataaccess.post_get import (
    PostGetResult,
    PostGetResultAuthor,
    NotFoundError as PostGetNotFoundError,
)
from dataaccess.post_list import PostListResult, PostListResultAuthor


client = TestClient(app)


@pytest.fixture(autouse=True)
def override_db_dependency():
    """Replace the DB dependency with a stub so tests do not hit Postgres."""

    async def _fake_conn() -> AsyncIterator[None]:
        yield None

    app.dependency_overrides[get_db_connection] = _fake_conn
    yield
    app.dependency_overrides.pop(get_db_connection, None)


@pytest.fixture(autouse=True)
def stub_generated_functions(monkeypatch):
    """Stub SnapSQL-generated functions so routers can be tested in isolation."""

    now = datetime(2025, 11, 13, 12, 0, 0)

    post_author = PostListResultAuthor(username="stub-author", full_name="Stub Author")
    list_result = PostListResult(
        post_id=10,
        title="Stub Post",
        content="Stub Content",
        author_id=1,
        published=True,
        view_count=5,
        created_at=now,
        updated_at=now,
        author=[post_author],
    )

    get_result = PostGetResult(
        post_id=list_result.post_id,
        title=list_result.title,
        content=list_result.content,
        author_id=list_result.author_id,
        published=list_result.published,
        view_count=list_result.view_count,
        created_at=list_result.created_at,
        updated_at=list_result.updated_at,
        author=[PostGetResultAuthor(username=post_author.username, full_name=post_author.full_name)],
    )

    comment_author = CommentListByPostResultAuthor(username="stub-commenter", full_name="Stub Commenter")
    comment_result = CommentListByPostResult(
        comment_id=99,
        post_id=list_result.post_id,
        author_id=2,
        content="Great post!",
        created_at=now,
        updated_at=now,
        author=[comment_author],
    )

    async def fake_post_create(conn, title, content, author_id, published, created_by):
        return PostCreateResult(
            post_id=123,
            title=title,
            content=content,
            author_id=author_id,
            published=published,
            view_count=0,
            created_at=now,
            updated_at=now,
            created_by=created_by,
            updated_by=created_by,
        )

    async def fake_post_get(conn, post_id):
        return get_result

    async def fake_post_list(conn, limit, offset):
        yield list_result

    async def fake_comment_create(conn, post_id, author_id, content):
        return CommentCreateResult(
            comment_id=comment_result.comment_id,
            post_id=post_id,
            author_id=author_id,
            content=content,
            created_at=now,
            updated_at=now,
        )

    async def fake_comment_list_by_post(conn, post_id):
        if post_id != comment_result.post_id:
            return
        yield comment_result

    monkeypatch.setattr(posts_router, "post_create", fake_post_create, raising=False)
    monkeypatch.setattr(posts_router, "post_get", fake_post_get, raising=False)
    monkeypatch.setattr(posts_router, "post_list", fake_post_list, raising=False)
    monkeypatch.setattr(comments_router, "comment_create", fake_comment_create, raising=False)
    monkeypatch.setattr(comments_router, "comment_list_by_post", fake_comment_list_by_post, raising=False)


class TestHealthEndpoints:
    """Test health check endpoints"""

    def test_root_endpoint(self):
        response = client.get("/")
        assert response.status_code == 200
        data = response.json()
        assert data["version"] == "1.0.0"

    def test_health_check(self):
        response = client.get("/health")
        assert response.status_code == 200
        assert response.json()["status"] == "healthy"


class TestPostEndpoints:
    """Test post-related endpoints that now call SnapSQL code."""

    def test_create_post_returns_response(self):
        response = client.post(
            "/posts?author_id=7",
            json={
                "title": "CLI Post",
                "content": "Body",
                "published": True,
            },
        )
        assert response.status_code == 201
        body = response.json()
        assert body["title"] == "CLI Post"
        assert body["author_id"] == 7
        assert body["created_by"] == 7

    def test_get_post_returns_author_metadata(self):
        response = client.get("/posts/10")
        assert response.status_code == 200
        body = response.json()
        assert body["author_username"] == "stub-author"
        assert body["post_id"] == 10

    def test_get_post_not_found(self, monkeypatch):
        async def _not_found(conn, post_id):
            raise PostGetNotFoundError("Record not found")

        monkeypatch.setattr(posts_router, "post_get", _not_found, raising=False)
        response = client.get("/posts/999")
        assert response.status_code == 404
        assert "Record not found" in response.text

    def test_list_posts_returns_data(self):
        response = client.get("/posts")
        assert response.status_code == 200
        body = response.json()
        assert isinstance(body, list)
        assert body[0]["author_username"] == "stub-author"


class TestCommentEndpoints:
    """Test comment-related endpoints"""

    def test_create_comment_returns_comment_response(self):
        response = client.post(
            "/comments?author_id=2",
            json={
                "post_id": 10,
                "content": "Nice!",
            },
        )
        assert response.status_code == 201
        body = response.json()
        assert body["post_id"] == 10
        assert body["author_id"] == 2

    def test_list_comments_returns_author_info(self):
        response = client.get("/comments/post/10")
        assert response.status_code == 200
        body = response.json()
        assert body[0]["author_username"] == "stub-commenter"
