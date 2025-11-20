import sys
from pathlib import Path

import pytest_asyncio

STUB_ROOT = Path(__file__).resolve().parent / "stubs"
if str(STUB_ROOT) not in sys.path:
    sys.path.insert(0, str(STUB_ROOT))

REPO_ROOT = Path(__file__).resolve().parents[5]
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))

TESTDATA_ROOT = REPO_ROOT / "testdata" / "appsample"
DDL_SQL = (TESTDATA_ROOT / "ddl.sql").read_text(encoding="utf-8")


@pytest_asyncio.fixture
async def sqlite_db():
    import aiosqlite

    conn = await aiosqlite.connect(":memory:")
    conn.row_factory = aiosqlite.Row
    await conn.executescript(DDL_SQL)
    await conn.commit()
    try:
        yield conn
    finally:
        await conn.close()
