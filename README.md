# Hoconenv

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
os.Getenv("DATABASE_URL")
```

If you're even lazier than that, you can simply import Hoconenv using a blank identifier, like so:

```go
import _ "github.com/ezrantn/hoconenv/autoload"
```

This way, you don't need to call `Load` explicitly. Just use `os.Getenv` to retrieve your variables.

> [!NOTE]
> If you're using the blank identifier approach, keep in mind that Hoconenv will use the default options. You cannot change these options in this case.

### Options

Hoconenv allows you to specify the following options:

- Continue loading despite errors.
- Override existing environment variables.
- Add a prefix for all environment variables (specific to a file).
- Define file patterns to include  (`.conf`, `.hocon`).

Below is the list of available fields for configuring options:

```go
IgnoreErrors    bool 
OverwriteEnv    bool
DefaultPrefix   string
IncludePatterns []string
```

By default, Hoconenv uses the following options:

```go
IgnoreErrors:    false,
OverwriteEnv:    true,
DefaultPrefix:   "",
IncludePatterns: []string{".conf", ".hocon"},
```

If you want to customize these options, you can do so as follows:

```go
// Get the default options
opts := hoconenv.DefaultOptions()

// For example, add a prefix
opts.DefaultPrefix = "APP"

// Load the configuration file with the specified options
err := hoconenv.LoadWithOptions(opts, "config.conf")
```

### Configuration File Format

Hoconenv supports the HOCON format with the following features:

- Comments: Use # or // for single-line comments.
- Nested Objects: Objects can be nested inside curly braces {}.
- Key-Value Pairs: Keys and values are defined using the = sign.
- Environment Variables: Configuration keys are converted to environment variables (uppercased and . replaced with _).

#### Example `application.conf`

```conf
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

- **APP_NAME=MyApp**
- **APP_DATABASE_HOST=localhost**
- **APP_DATABASE_PORT=5432**

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
