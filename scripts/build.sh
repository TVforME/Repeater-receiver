#!/bin/bash

set -e

# Check for root privileges
if [ "$(id -u)" -ne 0; then
    echo "This script must be run as root."
    exit 1
fi

# Input parameters
IP_ADDRESS=$1
GATEWAY=$2
NTP_POOL=$3

# change to match you use requirements.
USER="dvb"
SSH_DIR="/home/$USER/.ssh"
PRIVATE_KEY_PATH="$SSH_DIR/id_rsa"
PUBLIC_KEY_PATH="$SSH_DIR/id_rsa.pub"

if [ -z "$IP_ADDRESS" ] || [ -z "$GATEWAY" ] || [ -z "$NTP_SERVER" ]; then
    echo "Usage: $0 <IP_ADDRESS> <GATEWAY> <NTP_SERVER>"
    exit 1
fi

# Update and install required packages
apt update
apt upgrade -y

# Configure static IP address
echo "Configuring static IP address..."
cat <<EOF > /etc/netplan/01-netcfg.yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: no
      addresses:
        - $IP_ADDRESS/24
      gateway4: $GATEWAY
      nameservers:
        addresses:
          - 8.8.8.8
          - 8.8.4.4
EOF
netplan apply

# Configure NTP
echo "Configuring NTP..."
cat <<EOF > /etc/systemd/timesyncd.conf
[Time]
NTP=$NTP_SERVER
EOF
systemctl restart systemd-timesyncd

# Create SSH key pair if it doesn't exist
if [ ! -f "$PRIVATE_KEY_PATH" ]; then
    echo "Creating SSH key pair..."
    mkdir -p $SSH_DIR
    ssh-keygen -t rsa -b 4096 -f $PRIVATE_KEY_PATH -N ""
    chown -R $USER:$USER $SSH_DIR
    chmod 700 $SSH_DIR
    chmod 600 $PRIVATE_KEY_PATH
    chmod 644 $PUBLIC_KEY_PATH
fi

# Display the public key
echo "Public key to be added to the remote Raspberry Pi's authorized_keys:"
cat $PUBLIC_KEY_PATH

# Instructions for the user
echo "Please add the above public key to the remote Raspberry Pi's authorized_keys file."
echo "Run the following command on the Raspberry Pi:"
echo "cat <<EOF >> $SSH_DIR/authorized_keys"
cat $PUBLIC_KEY_PATH
echo "EOF"

# Download and install latest build tools
apt install -y build-essential linux-headers-$(uname -r) cmake pkg-config patchutils git wget netplan.io

# Clone and build TBS drivers OR
# https://github.com/janbar/tbs-dvb-driver
echo "Cloning and building TBS drivers..."
git clone https://github.com/tbsdtv/media_build.git
git clone --depth=1 https://github.com/tbsdtv/linux_media.git -b latest ./media
cd media_build
make dir DIR=../media
make allyesconfig
make -j4
sudo make install

# Clean up remove build files
rm -rf linux_media
rm -rf media_build

# Install dependencies for TSduck
echo "Installing dependencies for TSduck..."
apt install -y git cmake g++ pkg-config libcurl4-openssl-dev libpcap-dev

# Clone and build TSduck from source
echo "Cloning and building TSduck from source..."
locale-gen en_US.UTF-8
git clone https://github.com/tsduck/tsduck.git
cd tsduck
# Install basic as we are only using dvb receiver. No TX cards or protocols.
./scripts/install-prerequisites.sh NOJAVA=1 NODOXYGEN=1 NOPCSC=1 NODEKTEC=1 NOVATEK=1 NOHIDES=1 NOSRT=1 NORIST=1 NOTEST=1
make NOPCSC=1 NODEKTEC=1 NOVATEK=1 NOHIDES=1 NOSRT=1 NORIST=1 NOTEST=1 NOJAVA=1 -j4
make install
cd ..

# Clean up remove build files
rm -rf tsduck

# Install dvblast for testing dvb front end.
echo "Installing dvblast..."
apt install -y dvblast

# Install Go
echo "Installing Go..."
wget https://golang.org/dl/go1.20.3.linux-arm64.tar.gz
tar -C /usr/local -xzf go1.20.3.linux-arm64.tar.gz
echo "export PATH=\$PATH:/usr/local/go/bin" >> ~/.profile
source ~/.profile

# Verify Go installation
go version

# Set up Go workspace
mkdir -p ~/go/{bin,pkg,src}
echo "export GOPATH=\$HOME/go" >> ~/.profile
echo "export PATH=\$PATH:\$GOPATH/bin" >> ~/.profile
source ~/.profile

# Download and build Go packages from GitHub
echo "Downloading and building Go packages..."
go get -u github.com/TVforME/Repeater-receiver
cd ~/go/src/github.com/TVforME/Repeater-receiver
go build

echo "Setup complete!"


