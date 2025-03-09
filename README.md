# ragie-cli

A command line interface for importing various data formats into Ragie.

## Installation

### Using Homebrew (recommended)

```bash
# Add the Ragie tap
brew tap ragieai/tap

# Install ragie-cli
brew install ragie
```

### Manual Installation

1. Make sure you have Go 1.16 or later installed
2. Clone this repository
3. Run `go install` in the repository root

## Configuration

Set your Ragie API key as an environment variable:

```bash
export RAGIE_API_KEY=your_api_key_here
```

## Usage

### Import YouTube Data

```bash
ragie import youtube path/to/youtube.json [--dry-run] [--delay 2.0] [--partition your-partition]
```

### Import WordPress Data

```bash
ragie import wordpress path/to/wordpress.xml [--dry-run] [--delay 2.0] [--partition your-partition]
```

### Import ReadmeIO Data

```bash
ragie import readmeio path/to/readme.zip [--dry-run] [--delay 2.0] [--partition your-partition]
```

### Import Files from Directory

```bash
ragie import files path/to/directory [--dry-run] [--delay 2.0] [--partition your-partition]
```

The files importer will recursively scan the specified directory and import all non-empty files. Each file will be imported as a document with the following metadata:
- `source_type`: "files"
- `path`: The relative path from the import directory
- `extension`: The file extension
- `size`: The file size in bytes
- `mod_time`: The file's last modification time

### Clear All Documents

```bash
ragie clear [--dry-run] [--partition your-partition]
```

### Global Flags

- `--dry-run`: Print what would happen without making changes
- `--delay`: Delay between imports in seconds (default: 2.0)
- `--partition`: Specify a custom partition for your data (e.g., "production", "staging", "test")

## Development

1. Clone the repository
2. Run `go mod download` to install dependencies
3. Make your changes
4. Run `go build` to build the binary

## Testing

### Unit Tests

Run unit tests with:

```bash
go test ./...
```

### Integration Tests

Integration tests require a valid Ragie API key and will make actual API calls. To run integration tests:

```bash
export RAGIE_API_KEY=your_api_key_here
export INTEGRATION_TEST=true
go test ./integration_test -v
```

Note: Integration tests will create and delete test documents in your Ragie account. They clean up after themselves, but you may want to use a test account.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details. 