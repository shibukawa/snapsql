# Formatter Test Files

This directory contains test files for the SnapSQL formatter functionality.

## Files

- `test_format.sql` - Basic SQL file for testing SQL formatting
- `test_format.snap.sql` - SnapSQL template file with directives
- `test_format.md` - Markdown file with SQL code blocks
- `test_format.snap.md` - SnapSQL Markdown file with template syntax
- `test_format_backup.md` - Backup of original Markdown file for comparison

## Usage

These files can be used to test the formatter functionality:

```bash
# Format individual files
snapsql format testdata/formatter/test_format.sql
snapsql format testdata/formatter/test_format.md
snapsql format testdata/formatter/test_format.snap.md

# Format entire directory
snapsql format testdata/formatter/

# Format with write option
snapsql format -w testdata/formatter/test_format.sql

# Check formatting
snapsql format -c testdata/formatter/

# Show diff
snapsql format -d testdata/formatter/test_format.sql
```

## Expected Behavior

- **SQL files**: Keywords are uppercased, proper indentation with 4 spaces, trailing comma style
- **Markdown files**: SQL code blocks within ` ```sql ` are formatted, other content preserved
- **SnapSQL files**: Template directives (`/*# if */`, `/*= expr */`) are preserved and properly indented
