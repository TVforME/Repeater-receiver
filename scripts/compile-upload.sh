#!/bin/bash

#*******************************************************************************
#
#  Script to build dvb-receiver for pi CM4 64 and upload to pi via scp
#
#*******************************************************************************

# Save original Go environment variables
ORIGINAL_GOOS=$GOOS
ORIGINAL_GOARCH=$GOARCH

# Variables for cross-compilation
TARGET_GOOS=linux
TARGET_GOARCH=arm64
BINARY_NAME=dvb-receiver
CONFIG_FILE=config.yaml
UPLOAD_DIR=static # Replace with the actual directory name you want to upload
PI_USER=dvb
PI_HOST=192.168.0.74
REMOTE_DIR=/home/dvb/dvb-receiver


# Set Go environment variables for cross-compilation
export GOOS=$TARGET_GOOS
export GOARCH=$TARGET_GOARCH

# Function to check and install required packages
check_and_install_packages() {
    REQUIRED_PKG1="gcc-aarch64-linux-gnu"
    REQUIRED_PKG2="libc6-dev-arm64-cross"

    PKG_OK1=$(dpkg-query -W --showformat='${Status}\n' $REQUIRED_PKG1 | grep "install ok installed")
    PKG_OK2=$(dpkg-query -W --showformat='${Status}\n' $REQUIRED_PKG2 | grep "install ok installed")

    echo "Checking for $REQUIRED_PKG1 and $REQUIRED_PKG2..."

    if [ "" == "$PKG_OK1" ] || [ "" == "$PKG_OK2" ]; then
        echo "One or both required packages are missing. Installing..."
        sudo apt-get update
        sudo apt-get install -y $REQUIRED_PKG1 $REQUIRED_PKG2
    else
        echo "All required packages are installed."
    fi
}

# Check and install necessary cross-compilation tools
check_and_install_packages

# Compile the Go program
echo "Compiling $BINARY_NAME for $GOOS/$GOARCH..."
CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc go build -o $BINARY_NAME

# Check if the build was successful
if [ $? -ne 0 ]; then
    echo "Build failed. Restoring build to $ORIGINAL_GOARCH and $ORIGINAL_GOOS"
    export GOOS=$ORIGINAL_GOOS
    export GOARCH=$ORIGINAL_GOARCH
    exit 1
fi

# Upload the binary, config file, and directory to the Raspberry Pi
echo "Uploading $BINARY_NAME, $CONFIG_FILE, and $UPLOAD_DIR to $PI_USER@$PI_HOST:$REMOTE_DIR..."
scp $BINARY_NAME $PI_USER@$PI_HOST:$REMOTE_DIR/
scp $CONFIG_FILE $PI_USER@$PI_HOST:$REMOTE_DIR/
scp -r $UPLOAD_DIR $PI_USER@$PI_HOST:$REMOTE_DIR/

# Check if the uploads were successful
if [ $? -ne 0 ]; then
    echo "Upload failed. Exiting."
    exit 1
fi

echo "All files uploaded successfully."

# Reset Go environment variables to the original settings
export GOOS=$ORIGINAL_GOOS
export GOARCH=$ORIGINAL_GOARCH

echo "Go environment variables reset to original settings: GOOS=$GOOS, GOARCH=$GOARCH"


