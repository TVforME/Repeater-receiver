# What is Repeater-receiver?
Repeater-Receiver belongs to the Repeter project and is one of the key elements to my DATV [Repeater](https://github.com/TVforME/Repeater) Project.

Repeater-Receiver is a software version of a Set Top Box (STB) 
Most of it time will be listening on each of our repeater input frequencies waiting for a DVB-T/T2 and DVB-S/S2 signal.

VK3RGL is licenced for 1246MHz and 1255MHz in the 23cm amateur band with 1278MHz and 1287MHz inputs to be included in future.

# What does it do?
Repeater-Receiver is configured to operate in a specific way similar to either a Terrestrial DVB-T and Satellite DVB-S STB without any of the human interfaces such as remote control, front panel display, etc.  
A typical hardware STB (firmware) are usually closed sourced which doesn't allow for changes or experimentation with changing there operation suitable fo DATV repeater operation.

## Overview
The receiver is based around the TBS Technologies TBS6522 Quad multi-system DVB PCIe card although any other dvb adapter can be used with their drivers and firmware.
Both the DVB-T/T2 inputs are downconverted by 600MHz offset to place the 23cm frequencies in the common UHF broadcast band. The offset frequency is configured in the config.yaml file. 

DVB-S/S2 are received since the 23cm band is conveniently in the same IF range of the DVB-S/S2 receivers.

Each of the cards PCIe x1 are connected to the Compute Module 4 (CM4) via a daughter board. Unfortunately, the standard Raspberry Pi 4 PCIe lanes are pre-occupied by the on board usb switch and other peripherals which leaves the vanilla Pi4 not suitable for the project however, usable if USB dvb adapters are used.

2x TBS6522H Quad DVB boards allow for 8 individual adapters frontends to connect PCIe (x1) lane via a PCIe 2x (x1) expander board giving  4 x DVB-T/T2/T2-Lite and 4 x DVB-S/S2/S2X frontend receivers. Effectively 8 STB's.

## Here's a break down on how the receiver functions.

1. Once a signal is received, the TS is anaylsed for the first service in the MPTS or SPTS.
    
2. The frontend then analyses the first service for the signal modulation type.
   Using AUTO-T and AUTO-S Types in the config allows for both T/T2/T2-Lite and S/S2/S2X to be auto selected.
   
3. The TS is again anaylsed to determine the PCR, VIDEO and AUDIO PID's then begins sending the TS via RTP (relatime protocol) multicast out of the network port.
   I've tested using ffmpeg/ffplay, VLC and GStreamer and are able to receiver the RTP stream.
   
4. When any of the adaptor frontends are locked* repeater-receiver begins and emit a multicast UDP in json string which is polled in the repeater core.
   The repeater core acknowledges there is valid lock frontend and switches to the RTP stream. Once the UDP json string signals unlocked the connection assume lost.
   
5. Each configured adapter has it own OSD to show the frontend status and signal, SNR and BER level 0-100% and the service information on a OSD similar to a STB.
   The purpose is to superimpose the OSD onto any incoming video. The OSD is using SSE json events with html5, CSS and some Javascript.  My first attempt here too?

The source code is developed in Golang to levergage Go co-routines and Go's inbuilt http server including serval other cool packages the Go community have to offer.
I'm by no mean a Go programmer and this is my first DVB project using Go.
I've settled in using the TSduck tool kit for the underlaying engine at the moment however, there be a change to use GSteamer dvbsrc element once I master how to listen for the dvb messages from the gstreamer bus.
Go is GStreamer framework [ go-gst/go-gst](https://github.com/go-gst/go-gst) is missing the functions required to handle DVB PSI at the moment.

## Code Quality..
I'm particular and strive to improve on what is offered. 
Please support the project. If your a Go Guru and can offer improvements, please reachout. Checkout my TODO list for future work and enhancements.

## Screen shots of receiver running on a Linux i5 Gen 9 Laptop using 2 x USB DVB-T adapters.

Take note the inset video is a GStreamer pipeline receiving the RTP TS from the selected adapter and is not part of the code base. VLC can do the same using rtp://239.255.0.1:5004 or what ever your multicast address you assign.

<img src="/docs/images/Livescreenshot-adapter-osd-gst.jpg" width="65%">

<img src="/docs/images/Screenshot_2024-08-05_System_Monitor.png" width="45%">

Each adapter's frontend are polled in read only mode using ioctl calls.  Direct ioctl calls greatly speed up the response and has avoided the additional overhead in translating the stats values comming in through stdio.

The underlying adapter tuning and demux in anaylsing of services to select pcr, video and audio PID's is reliant on [TSduck](https://tsduck.io/download/tsduck/) 

TSduck and dvbapi are a required dependancies for receiver is operate.

The code operates on a simple state machine with 4 states:-

1. Listening
2. Analysing
3. Streaming
4. Stopping

All settings are configured in a config.yaml file and read at startup.

## Architectures AMD64 and ARM64.
Version 1.0.1 is operating on Linux Ubuntu 24.04 and Ubuntu (headless) on Raspberry Pi.
Ulimately, Receiver is in development to be ported to PiCore64 for the purpose of running entirely in RAM to avoid SSD corruption from premature power removal.


## Building Repeater Reciever for amd64 and arm64 (Raspberry Pi)

Below is the dir tree showing the files
```bash
Repeater-receiver/
├── docs/
├── repo/
├── src/
│    ├─────── static/ 
│    ├── ...     ├── adapter.html 
│    ├── ...     ├── monitor.html
│    ├── ...     ├── root.html      
│    └── ...     └── style.css
│ 
├── config.yaml
├── main.go
├── go.mod
├── go.sum
└── TODO
```

###  Dependencies and build system:

Install the TSduck tool kit, your adapter drivers and if not already on your distro, the dvbapi

**** More to be added here  *****


To build your Go application for both AMD64 and ARM64 architectures, including ARM64 for Raspberry Pi, follow these detailed steps:

### Step 1: Install Go
First, install the Go programming language on your machine.
Download and Install Go on your development machine. At the time of writing the latest was version 1.22.6
Adjust to suit the latest version. Best to check for the latest version offeredhere https://go.dev/dl/ and go with that.

```bash
wget https://golang.org/dl/go1.22.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.6.linux-amd64.tar.gz
```
Add Go to your PATH:

```bash
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.profile
source ~/.profile
```
Verify Installation: 

```bash
go version
```
Confirm that the command prints the installed version of Go such as v1.22.6
Once Go is installed.. We are good to Go!

### Step 2: Clone the Repository:

```bash
git clone https://github.com/TVforME/Repeater-receiver.git
cd Repeater-receiver
```
```bash
go mod tidy
```

### Step 3: Build the Repository:

### Build for AMD64:
```bash
GOARCH=amd64 GOOS=linux go build -o repeater-receiver-amd64
```
### Build for ARM64:
```bash
GOARCH=arm64 GOOS=linux go build -o repeater-receiver-arm64
```
Ensure the binaries repeater-receiver-amd64 and repeater-receiver-arm64 are created in your root directory.

### Step 4: Modify the config file.
Modify the config.yam file to suit your dvb apaters and frequency, bandwitdh and sysmbol-rate values.

### Step 5: Run the application on our platform

AMD64 
```bash
./repeater-receiver-amd64
```
 ARM64 (for me raspberry Pi CM4)
```bash
./repeater-receiver-arm64
```
# Below this line is proposed to porting to piCore64
# =============================================

## Using piCore64 on a Raspberry Pi Compute Module 4 (CM4) 
piCore offers significant advantages with reliability and extending the life of SD card/USB/EMMC lifespan. piCore offers far beeter performance over similiar Linux OS such as Ubuntu core. core22 and raspberrianLite. I've been satisfied piCore to be a key contributor to the success so far. 

## piCore64?
piCore64 is a variant of [Tiny Core Linux](http://tinycorelinux.net/) is designed specifically for both 32-bit and 64-bit systems. It's extremely lightweight and operates completely in RAM. The initial boot process rootfs, OS image is loaded into memory. Did I say piCore64 runs entirely from RAM from there on. Raspberrian can be configured to run RO (Read Only) however, requires additional work to configure zswap and log rotation to be workable. 


## Benefits of Running Linux from RAM are:

### 1. Reduction of Write Operations:
Traditional Linux distributions write data frequently to their storage medium like SSD drives, SD cards, USB sticks, or EMMC etc. Each write operation potentially shortens the lifespan of these storage devices due to wear and tear on the flash cells. piCore64 minimises this risk entirely by operating in RAM, thus significantly reducing the number of write operations to flash devices. Even swapfile and logs use RAM.

### 2. Increased System Performance:
RAM is significantly faster than most forms of persistent storage, particularly SD cards and EMMC. By operating from RAM, piCore64 ensures that the system can run applications and processes much faster, which is crucial for real-time mission critical applications. Obviosly, any code memory leaks overtime is likely to slowly chew up 4Gb of RAM therefore it's imperitive applications written in C and C++ need to free resources correcly. 

A nightly reboot feature maybe a solution for memory leak.. A simple bash script could routinely watch memory capacity and schedule a reboot during early morning intervals.

### 3. Enhanced Reliability and Stability:
As in previous dot points, the risk of file system corruption due to unexpected power failures during a filesystem write is eliminated. This aspect alone is particularly important for systems that may to be deployed in remote or less accessible locations, where providing maintenance can be challenging.

### 4. Simplified System Maintenance:
With fewer writes to the storage device, the overall system maintenance is reduced. There's is really no need to continually update software reduces the concern about data integrity issues once to power goes off at the repeater site. Restoring power reloads a fresh copy into RAM and off it goes. Its easy enough to run the `picore_image_build.sh` script to use a later version of Linux kernel. Read on more to come.. 

### 5. 24/7/365 Benifits:
Developing the repeater receiver code on piCore64 ensures efficient and reliable performance in a broadcast environment.

The operating system's lightweight ensures that most of the Raspberry Pi’s 3 Cores are busy in handling the communications with PCI DVB adaptor/s as well as juggling multiple instances of DVBlast rather than spending cycles managing Snaps / apt triggers and many deamons running intermittantly in the background as with traditional Linux OS's. 

Additionally, the ephemeral nature of RAM-based systems means that any configuration changes or temporary data are reset upon reboot, which can help in maintaining a consistent state across power cycles. Despite RAM use files, piCore has facilities to make settings persistant by running simple scripts. I'm only new to piCore however, there are plently knowledgable people willing to help on the tinylinux forum for any issues or hurdles I've come across so far.

## Design Considerations
The receiver operates independently of the repeater for several reasons:

1. **Reduce Overhead**: This approach reduces overhead in the repeater Core design to manage dvb access. My concept is for all the RF to be kept in one box. Motherboard running at GHz clock speeds spells interference and potential noise issues if happened to have low level RF signals coaxial cables draped over motherboard componets.
2. Communication is through the 1Gbs Ethernet interface using RTP/UDP/HTTP with SSH for remote access.  Theoretically, the receive could be located remotely from the core!
3. **Configurable**: Receiver can be configured for different PCIe DVB cards to receive both DVB-S/S2 or DVB-T/T2 per frequency. DATV uses both DVB-S/S2 and DVB-T/T2 for terrestrial experimentation.
4. **Reduce Point of Failure**: The receiver is constructed to fit in a hard disk bay of the repeater ATX 4RU chassis. A duplicate receiver can be simply exchanged to facilitate "upgrades and features" in 5 minutes making servicability key importance at a repeater site.

## Hardware
- Raspberry Pi Compute Module 4 (CM4004032): 4GB RAM, 32GB eMMC, without WiFi to reduce RF at the site.
- I2C OLED Display: Used for fontpanel display to show status of the receiver.
- Modify TBS-6522H from 75 Ohm F-type to 50 Ohm SMA with 1:5 impedance balun. PCB to be designed to accomidate SMA's and baluns
## Picture below showing F-Type connectors removed ready for SMA and 1:5 balun board daughter board

<img src="/docs/images/TBS-6522H-noFtypes.jpg" width="25%">
  
## Software details
- **Operating System**: piCore64 version 14.1.0
- **Aplication**: Entirely in Go code move to => deamon in future coded in Go.
- **Build Script**: Script for setting up necessary v4ldvb, TBS drivers, a few tools such as pcitools and of coaurse the TSduck tool kit.
- I'll develop a tcz package to load on startup with all dependancies included.  That's the plan.

