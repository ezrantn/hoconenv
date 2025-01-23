# Hoconenv

![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/ezrantn/hoconenv/go.yml)
![GitHub License](https://img.shields.io/github/license/ezrantn/hoconenv)
![GitHub last commit](https://img.shields.io/github/last-commit/ezrantn/hoconenv)

Hoconenv is a Go library for loading [HOCON (Human-Optimized Config Object Notation)](https://docs.spongepowered.org/stable/en/server/getting-started/configuration/hocon.html) configuration files into environment variables. It supports comments, nested structures, and includes, providing an easy-to-use interface for configuring applications in Go.

## Features

- Parse HOCON configuration files.
- Support for environment variables.
- Handles comments (both # and //).
- Supports nested objects.
- Supports file inclusion (include directive).
- Automatically converts keys to environment variable format.

## Installation

To install Hoconenv, run the following command:

```bash
go get github.com/ezrantn/hoconenv
```

## Usage

### Quick Start

```go
// ...

//  Load default configuration file (application.conf by default)
err := hoconenv.Load()

// Or load specific file
err := hoconenv.Load("config.conf")

// Access via environment variables
os.Getenv("database.url")
```

If you're even lazier than that, you can simply import Hoconenv using a blank identifier, like so:

```go
import _ "github.com/ezrantn/hoconenv/autoload"
```

This way, you don't need to call `Load` explicitly. Just use `os.Getenv` to retrieve your variables.

### Prefix

Hoconenv supports the use of a prefix. The global prefix applies to all environment variables set by the package.

To set a global prefix, use:

```go
// Set a global prefix
hoconenv.SetPrefix("prod")

// After setting this prefix, all variables accessed should use the "prod" prefix:
// For example:
os.Getenv("prod.database.url")
os.Getenv("prod.database.host")
```

### Configuration File Format

Hoconenv supports the HOCON format with the following features:

- Comments: Use # or // for single-line comments.
- Nested Objects: Objects can be nested inside curly braces {}.
- Key-Value Pairs: Keys and values are defined using the = sign.
- Environment Variables: Configuration keys are converted to environment variables (**lowercase and separated by `.`**).

#### Example `application.conf`

```json
# This is a comment
app {
    name = MyApp
    database {
        host = localhost # Inline comment
        port = 5432
    }
}

# Another comment
include "additional_config.conf"
```

The above example will be parsed and converted into the following environment variables:

- **app.name = MyApp**
- **app.database.host = localhost**
- **app.database.port = 5432**

If the `include` directive is used, it will recursively load the included file (`additional_config.conf` in this case).

### File Inclusion

Hoconenv supports including other configuration files within the main configuration using the `include` directive.

```bash
include "other_config.conf"
```

This will automatically load the file `other_config.conf` and parse its contents.

## License

This tool is open-source and available under the [MIT License](https://github.com/ezrantn/hoconenv/blob/main/LICENSE).

## Contributing

Contributions are welcome! Please feel free to submit a pull request. For major changes, please open an issue first to discuss what you would like to change.
