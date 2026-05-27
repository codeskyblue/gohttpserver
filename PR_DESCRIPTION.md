# Feature: Add file or directory movement function

This PR implements the file and directory move functionality as requested in #108.

## Changes

- Added `/-/move` API endpoint (POST method).
- Implemented `hMove` handler in `httpstaticserver.go` to handle move operations.
- Refactored `getRealPath` to use a helper `resolvePath` method for consistent path resolution.
- Added comprehensive permissions and existence checks.
- Implemented overwrite logic for existing destinations.
- Added comprehensive unit tests in `move_test.go` covering:
    - Moving files and directories.
    - Moving to new (non-existent) parent directories.
    - Handling conflicts (destination already exists).
    - Handling non-existent source.
    - Permission checks.

## API Usage

**Endpoint:** `POST /-/move` 

**Parameters:**
- `src`: Source path (relative to root)
- `dst`: Destination path (relative to root)
- `overwrite`: Set to `true` to overwrite destination if it exists (optional, default: `false`)

**Responses:**
- `200 OK`: Success `{"success": true}` 
- `400 Bad Request`: Missing `src` or `dst` 
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Source does not exist
- `409 Conflict`: Destination already exists
- `500 Internal Server Error`: File system error

## Verification

The feature was verified with a local integration test suite covering file/directory movement, overwrite logic, and permission checks.

**Manual Verification:**
```bash
# Move file
curl -X POST -d "src=/test.txt" -d "dst=/test_moved.txt" http://localhost:8000/-/move

# Move dir
curl -X POST -d "src=/dir" -d "dst=/dir_moved" http://localhost:8000/-/move

# Overwrite
curl -X POST -d "src=/a.txt" -d "dst=/b.txt" -d "overwrite=true" http://localhost:8000/-/move
```
