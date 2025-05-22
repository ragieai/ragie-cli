# Integration Tests

This directory contains integration tests for the Ragie CLI import functionality.

## Running Tests

### Unit Tests

Unit tests can be run without any external dependencies:

```bash
go test ./cmd -v
```

These tests verify the logic of the `--force` flag without making actual API calls.

### Integration Tests

Integration tests require a real API connection and should only be run with a test environment:

```bash
# Set required environment variables
export RAGIE_API_KEY="your-test-api-key"
export INTEGRATION_TEST="true"

# Run all integration tests
go test ./integration_test -v

# Run specific test
go test ./integration_test -v -run TestForceFlag
```

## Force Flag Tests

The `--force` flag tests are integrated into each importer's test file and verify the following behavior:

### Without `--force` flag:
- Documents with existing external IDs are skipped
- Only non-existing documents are imported
- Warning messages are displayed for skipped documents

### With `--force` flag:
- Documents are imported even if they have existing external IDs
- Creates duplicate documents with the same external ID
- All import types support the force flag

## Test Coverage

Each importer has its own force flag test:

- **YouTube**: `TestYouTubeImportForce` in `youtube_test.go` ✅
- **Files**: `TestFilesImportForce` in `files_test.go` ✅
- **WordPress**: `TestWordPressImportForce` in `wordpress_test.go` ✅
- **ReadmeIO**: `TestReadmeIOImportForce` in `readmeio_test.go` ✅  
- **Zip**: `TestZipImportForce` in `zip_test.go` ✅

## Test Data

Integration tests create temporary test data and clean up after themselves. Each test:

1. Creates unique test data with predictable external IDs
2. Cleans up any existing test documents before starting
3. Runs the import without force (should succeed)
4. Runs the import again without force (should skip)
5. Runs the import with force (should create duplicates)
6. Verifies the correct number of documents exist
7. Cleans up all test documents

## Safety

These tests are designed to be safe for testing environments:
- Uses unique external IDs that won't conflict with real data
- Cleans up all test documents after each test
- Skips automatically unless `INTEGRATION_TEST=true` is set
- Uses temporary files that are automatically cleaned up 