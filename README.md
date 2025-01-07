# MX - Microservice Gateway

MX is a lightweight and flexible microservice gateway framework.

## Installation

Install the MX CLI tool:

```bash
go install github.com/hysios/mx/cmd/mx@latest
```

## Commands

### Generate Commands

Generate new service or gateway:

```bash
# Generate a new service
mx gen service --name user --pkg-name github.com/example/user

# Generate a new gateway
mx gen gateway --pkg-name github.com/example/gateway

# Add a new service to existing project
mx gen add --name payment --pkg-name github.com/example/payment
```

### Gateway Commands

Run the gateway server:

```bash
mx gateway --addr :8080
```

### Configuration Commands

MX supports multiple configuration backends. Here's how to use them:

#### Setup Redis Backend

```bash
consul services register -name="mx.Config" \
  -meta=service_type=config_provider \
  -meta=targetURI=redis://127.0.0.1:6379/mx.config \
  -address=127.0.0.1 \
  -port=6379
```

#### Config Management

```bash
# Set a config value
mx config set -key=key=value

# Get a config value
mx config get -key=key

# Get a config value (quiet mode)
mx config get -key=key --quite

# View all configs
mx config cat

# Update config with JSON data
mx config update --data='{"key": "value"}'

# Update config from JSON file
mx config update --data=@/path/to/file.json
```

## Documentation

For Chinese documentation, please see [README_CN.md](README_CN.md)

## License

MIT License


