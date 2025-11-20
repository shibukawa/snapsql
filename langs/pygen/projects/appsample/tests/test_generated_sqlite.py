import pytest

from testdata.appsample.generated_python import (
    AccountGet,
    AccountList,
    AccountUpdate,
    UpdateAccountStatusConditional,
)


@pytest.mark.asyncio
async def test_account_get_returns_existing_row(sqlite_db):
    await sqlite_db.execute(
        "INSERT INTO accounts (id, name, status) VALUES (?, ?, ?)",
        (1, "Alice", "active"),
    )
    await sqlite_db.commit()

    async with sqlite_db.cursor() as cursor:
        result = await AccountGet(cursor, 1)

    assert result.id == 1
    assert result.name == "Alice"
    assert result.status == "active"


@pytest.mark.asyncio
async def test_account_list_yields_results(sqlite_db):
    await sqlite_db.executemany(
        "INSERT INTO accounts (id, name, status) VALUES (?, ?, ?)",
        [
            (1, "Alpha", "active"),
            (2, "Beta", "inactive"),
        ],
    )
    await sqlite_db.commit()

    async with sqlite_db.cursor() as cursor:
        results = []
        async for row in AccountList(cursor):
            results.append(row)

    assert [row.id for row in results] == [2, 1]
    assert results[0].name == "Beta"
    assert results[1].status == "active"


@pytest.mark.asyncio
async def test_account_update_returns_updated_rows(sqlite_db):
    await sqlite_db.execute(
        "INSERT INTO accounts (id, name, status) VALUES (?, ?, ?)",
        (1, "Alpha", "active"),
    )
    await sqlite_db.commit()

    async with sqlite_db.cursor() as cursor:
        rows = []
        async for row in AccountUpdate(cursor, 1, "archived"):
            rows.append(row)

    assert len(rows) == 1
    assert rows[0].id == 1
    assert rows[0].status == "archived"


@pytest.mark.asyncio
async def test_update_account_status_conditional_updates_conditionally(sqlite_db):
    await sqlite_db.executemany(
        "INSERT INTO accounts (id, name, status) VALUES (?, ?, ?)",
        [
            (1, "Alpha", "active"),
            (2, "Beta", "active"),
        ],
    )
    await sqlite_db.commit()

    async with sqlite_db.cursor() as cursor:
        updated = await UpdateAccountStatusConditional(cursor, "inactive", 1, True)

    assert updated == 1

    async with sqlite_db.cursor() as cursor:
        updated_none = await UpdateAccountStatusConditional(cursor, "inactive", 999, True)

    assert updated_none == 0
