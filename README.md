# Web Traffic Generator

The `webtraffic` tool is a CLI-based implementation designed to simulate "organic" web browsing behaviour for demonstration or testing purposes.

## Features

- Recursive web surfing with configurable depth
- Configurable delay between requests
- Blacklist support to avoid unwanted domains
- Detailed logging with verbose option
- Easily configurable via YAML file or command line flags
- Automatic handling of rate limiting (HTTP 429 responses)

## Installation

### Prerequisites

- Go 1.16 or higher

### Building from source

1. Clone the repository: ... repo url goes here ...
2. Build the binary: `go build -o webtraffic`

## Configuration

The tool can be configured using a YAML file or command line flags. 
By default, it looks for a `.webtraffic.yaml` file in your home directory.

### Sample Configuration File

Create a file named `.webtraffic.yaml` in your home directory with the following content:

```yaml
verbose: false
max_depth: 10
min_depth: 3
max_wait: 10
min_wait: 5
root_urls:
- https://www.example.com
- https://www.another-example.com
blacklist:
- facebook.com
- pinterest.com
user_agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36"
```

### Command-line Flags

- config: Specify a custom config file path
- verbose: Enable verbose logging
- max-depth: Set maximum browsing depth
- min-depth: Set minimum browsing depth
- max-wait: Set maximum wait time between requests
- min-wait: Set minimum wait time between requests

## Usage

Run the tool using the following command:

```ignorelang
../webtraffic
```

To use custom configuration:

```ignorelang
./webtraffic --config /path/to/custom/config.yaml
```

To override specific settings:

```ignorelang
./webtraffic --verbose --max-depth 15
```

## License

This project is licensed under the [MIT License](https://opensource.org/license/mit).

## Disclaimer

This tool is for educational and testing purposes only.
The authors are not responsible for any misuse or damage caused by this tool.

## Additional Notes
The idea to build this tool has been directly taken from the [ReconInfoSec Github account](https://github.com/ReconInfoSec/web-traffic-generator).
