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

### Basic Usage

You can load a configuration file using the `Load` function. If no file path is provided, it will attempt to load the default `application.conf` file.

#### Default Behavior (Load from `application.conf`)

```go
err := hoconenv.Load()
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

// Fetch an environment variable
appName := os.Getenv("APP_NAME")
fmt.Println("App Name:", appName)
```

#### Custom File Path

If you want to specify a custom configuration file path, pass the file path as an argument:

```go
hoconenv.Load("path/to/your/config.conf")
```

#### Set Prefix

If you wish to have a prefix in your environment. You can do that with:

```go
hoconenv.SetPrefix("PRODUCTION")

// Now if you call any variables inside your config, it will have "PRODUCTION" prefix

err := hoconenv.Load()
if err != nil {
    log.Fatalf("Failed to load config: %v", err)
}

appName := os.Getenv("PRODUCTION_APP_NAME")
fmt.Println(appName)
```

#### For the Lazy

If you're as lazy as I am, you can import Hoconenv as a blank identifier. This way, you don't need to explicitly call the `Load` method. Here's how:

```go
import (
    _ "github.com/ezrantn/hoconenv/autoload"
)
```

It will work the same as the rest of the code. You can access your environment variables like this:

```go
os.Getenv("APP_NAME")
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
