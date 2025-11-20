import httpx
import pytest


@pytest.mark.asyncio
async def test_live_server_health_endpoint(live_server_url):
    async with httpx.AsyncClient() as client:
        response = await client.get(f"{live_server_url}/health", timeout=5.0)

    assert response.status_code == 200
    assert response.json() == {"status": "healthy"}


@pytest.mark.asyncio
async def test_live_server_root_endpoint(live_server_url):
    async with httpx.AsyncClient() as client:
        response = await client.get(live_server_url + "/", timeout=5.0)

    assert response.status_code == 200
    payload = response.json()
    assert payload["message"].startswith("Welcome to Blog API")
    assert payload["version"] == "1.0.0"


@pytest.mark.asyncio
async def test_live_server_users_endpoint_not_implemented(live_server_url):
    async with httpx.AsyncClient() as client:
        response = await client.get(f"{live_server_url}/users", timeout=5.0)

    assert response.status_code == 501
