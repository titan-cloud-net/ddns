# DDNS - Dynamic DNS for Cloudflare

A lightweight service written in Go that automatically updates Cloudflare DNS A records based on your current public IP address. Perfect for home servers, development environments, or any scenario where your public IP address may change dynamically (e.g., via DHCP).

## Overview

This service monitors your public IP address and automatically updates the corresponding DNS A record in Cloudflare whenever it changes. It leverages the official [Cloudflare Go SDK](https://github.com/cloudflare/cloudflare-go) to interact with the Cloudflare API.

## Features

- **Automatic IP Detection**: Monitors your public IP address for changes
- **Dynamic Updates**: Automatically updates Cloudflare DNS A records when IP changes are detected
- **Cloudflare Integration**: Uses the official Cloudflare Go SDK for reliable API interactions
- **Lightweight**: Minimal resource footprint, suitable for running on constrained devices
- **DHCP-Friendly**: Designed to handle dynamic IP assignments from ISPs
- **Simple Configuration**: Easy setup via environment variables

## Use Cases

- **Home Servers**: Keep your home server accessible via a consistent domain name even when your ISP assigns a new IP
- **Development Environments**: Access your development server remotely without worrying about IP changes
- **IoT Devices**: Maintain connectivity to IoT devices with dynamic IP addresses
- **Small Business**: Cost-effective DNS management for small businesses with dynamic IP assignments

## Prerequisites

- Go 1.21 or higher (for building from source)
- A Cloudflare account with API access
- A domain managed by Cloudflare
- Internet connectivity to detect public IP address

## Installation

### From Source

```bash
git clone https://github.com/titan-cloud-net/ddns.git
cd ddns
go build -o ddns ./cmd
```

### Using Go Install

```bash
go install github.com/titan-cloud-net/ddns@latest
```

## Configuration

The service is configured using environment variables:

- `CLOUDFLARE_API_TOKEN`: Your Cloudflare API token with DNS edit permissions
- `CLOUDFLARE_ZONE_ID`: The Zone ID for your domain
- `DNS_RECORD_NAME`: The DNS record name to update (e.g., `home.example.com`)
- `CHECK_INTERVAL`: How often to check for IP changes in seconds (default: `300`)

## Usage

### Running the Service

```bash
export CLOUDFLARE_API_TOKEN="your-api-token"
export CLOUDFLARE_ZONE_ID="your-zone-id"
export DNS_RECORD_NAME="home.example.com"
export CHECK_INTERVAL="300"

./ddns
```

### Running as a Systemd Service

First, optionally create a dedicated user for running the service:

```bash
sudo useradd -r -s /bin/false ddns
```

Create a systemd service file at `/etc/systemd/system/ddns.service`:

```ini
[Unit]
Description=Dynamic DNS Service for Cloudflare
After=network.target

[Service]
Type=simple
User=ddns
Environment="CLOUDFLARE_API_TOKEN=your-api-token"
Environment="CLOUDFLARE_ZONE_ID=your-zone-id"
Environment="DNS_RECORD_NAME=home.example.com"
Environment="CHECK_INTERVAL=300"
ExecStart=/usr/local/bin/ddns
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

> **Note**: If you don't create a dedicated user, you can use an existing system user or remove the `User=ddns` line to run as root (not recommended for security).

Enable and start the service:

```bash
sudo systemctl enable ddns
sudo systemctl start ddns
sudo systemctl status ddns
```

### Running with Docker

```bash
docker build -t ddns .
docker run -d \
  --name ddns \
  --network host \
  -e CLOUDFLARE_API_TOKEN="your-api-token" \
  -e CLOUDFLARE_ZONE_ID="your-zone-id" \
  -e DNS_RECORD_NAME="home.example.com" \
  -e CHECK_INTERVAL="300" \
  ddns
```

## How It Works

1. **Initialization**: On startup, the service reads environment variables and validates Cloudflare credentials
2. **IP Detection**: Queries an external service to determine the current public IP address
3. **DNS Check**: Retrieves the current DNS A record value from Cloudflare
4. **Comparison**: Compares the detected public IP with the DNS record
5. **Update**: If they differ, updates the DNS A record via the Cloudflare API
6. **Monitoring**: Waits for the configured interval and repeats the process

> **Note**: The service automatically detects your public IP address, making it suitable for environments behind NAT (typical home/office router setups).

## Getting Your Cloudflare Credentials

### API Token

1. Log in to your Cloudflare dashboard
2. Go to **My Profile** > **API Tokens**
3. Click **Create Token**
4. Use the **Edit zone DNS** template or create a custom token with the following permissions:
   - Zone / DNS / Edit
5. Select the specific zone (domain) you want to manage
6. Copy the generated token

### Zone ID

1. Log in to your Cloudflare dashboard
2. Select your domain
3. Scroll down to the **API** section on the right sidebar
4. Copy the **Zone ID**

## Troubleshooting

### Service fails to start

- Verify your Cloudflare API token has the correct permissions
- Check that the Zone ID matches your domain
- Ensure the DNS record exists in Cloudflare (create it manually if needed)
- Verify all required environment variables are set

### IP address not updating

- Check that you have internet connectivity
- Review service logs for error messages
- Ensure your Cloudflare account has sufficient API rate limits
- Verify the DNS record name matches exactly (case-sensitive)

### Permission errors

- Ensure environment variables are accessible to the service user
- When running as systemd service, verify the user has necessary permissions

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Running Tests

```bash
go test ./...
```

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Cloudflare Go SDK](https://github.com/cloudflare/cloudflare-go) for the excellent API client library
- The Go community for their outstanding tools and libraries

## Support

For issues, questions, or contributions, please use the [GitHub Issues](https://github.com/titan-cloud-net/ddns/issues) page.
