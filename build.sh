#!/bin/bash

# Exit on any error
set -e

VERSION="1.0.1"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/repeater-receiver"

# Function to check if a command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo "$1 is required but not installed."
        return 1
    fi
}

# Function to install dependencies
install_dependencies() {
    echo "Checking and installing dependencies..."
    
    # Check if running as root
    if [ "$EUID" -ne 0 ]; then 
        echo "Please run with sudo to install dependencies"
        exit 1
    }

    # Update package lists
    apt-get update

    # Install TSDuck and DVB API
    apt-get install -y tsduck dvb-tools

    # Additional dependencies can be added here
    echo "Dependencies installed successfully"
}

# Check for Go installation and version
check_go_version() {
    if ! check_command go; then
        echo "Go is not installed. Installing Go..."
        wget https://go.dev/dl/go1.22.6.linux-amd64.tar.gz
        sudo tar -C /usr/local -xzf go1.22.6.linux-amd64.tar.gz
        echo "export PATH=\$PATH:/usr/local/go/bin" >> ~/.profile
        source ~/.profile
    fi

    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    if [ "$(printf '%s\n' "1.22" "$GO_VERSION" | sort -V | head -n1)" = "1.22" ]; then
        echo "Go version $GO_VERSION detected - OK"
    else
        echo "Go version must be 1.22 or higher"
        exit 1
    fi
}

# Build function
build() {
    local arch=$1
    local output_name="repeater-receiver-$arch"
    
    echo "Building for $arch architecture..."
    GOARCH=$arch GOOS=linux go build -o $output_name
    
    if [ $? -eq 0 ]; then
        echo "Build successful for $arch"
        chmod +x $output_name
    else
        echo "Build failed for $arch"
        exit 1
    fi
}

# Install function
install() {
    local arch=$1
    local binary_name="repeater-receiver-$arch"
    
    echo "Installing for $arch..."
    
    # Create necessary directories
    sudo mkdir -p $INSTALL_DIR
    sudo mkdir -p $CONFIG_DIR
    sudo mkdir -p $CONFIG_DIR/static
    
    # Install binary
    sudo install -m 755 $binary_name $INSTALL_DIR/repeater-receiver
    
    # Install configuration and static files
    sudo install -m 644 config.yaml $CONFIG_DIR/
    sudo cp -r src/static/* $CONFIG_DIR/static/
    
    echo "Installation completed for $arch"
}

# Main script execution
echo "Repeater-receiver build script v$VERSION"

# Check if help is needed
if [ "$1" = "-h" ] || [ "$1" = "--help" ]; then
    echo "Usage: $0 [--install-deps] [--amd64] [--arm64] [--all]"
    echo "  --install-deps : Install system dependencies"
    echo "  --amd64       : Build for AMD64 architecture"
    echo "  --arm64       : Build for ARM64 architecture"
    echo "  --all         : Build for all architectures"
    exit 0
fi

# Process arguments
if [ "$1" = "--install-deps" ]; then
    install_dependencies
    exit 0
fi

# Check Go installation
check_go_version

# Initialize modules
go mod tidy

# Determine build targets
if [ "$1" = "--amd64" ] || [ "$1" = "--all" ]; then
    build "amd64"
    install "amd64"
fi

if [ "$1" = "--arm64" ] || [ "$1" = "--all" ]; then
    build "arm64"
    install "arm64"
fi

if [ $# -eq 0 ]; then
    # Default to building for current architecture
    CURRENT_ARCH=$(uname -m)
    if [ "$CURRENT_ARCH" = "x86_64" ]; then
        build "amd64"
        install "amd64"
    elif [ "$CURRENT_ARCH" = "aarch64" ]; then
        build "arm64"
        install "arm64"
    else
        echo "Unsupported architecture: $CURRENT_ARCH"
        exit 1
    fi
fi

echo "Build process completed successfully!"
