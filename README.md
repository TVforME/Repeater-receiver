# Repeater-receiver
Receiver for the DATV Repeater Project..

Receiver simlulates a STB (Set Top Box) without the remote control and stuff we don't need.
Receiver operates independantly and listens for a valid TS on configured frequencies. Once a signal is received, the TS is anaylsed for the first service in the MPTS or SPTS and then reconfigures the demux to pass the first service onto the streamer. The streamer forwards the SPTS via RTP (relatime protocol) via multicast for the repeater core to decode. In effect, Receiver could be used as a simple headend.

## Overview
The receiver is based around the TBS Technologies TBS6522 Quad multi-system DVB PCIe card on a Compute Module 4 (CM4) on a AliExpress daughter board effectively exposes the CM4's PCIe channel. Unfortunately, the standard Raspberry Pi 4 has its PCIe lanes occupied for usb switch and other peripherals which leaves the use of USB DVB adapter the only solution. 
Using USB DVB adapters comes with caveats-

1. Limitited to how many usb ports available on your system. 
2. Order of usb enumerations can be problematic. /dvb/adapter0 adapter1 aren't always logical 0,1,2 etc
3. A solution is to bind the adapter 'Name' to usb port.

The CM4's PCIe support allows for 8 individual adaptors to connect PCIe (x1) lane via a PCIe 2x (x1) expander board.  

The source code is developed in Golang to levergage go routines and go's built in http server to host an OSD display to show each adapter and monitor page for CPU/RAM/Network usage rates.

<img src="/docs/images/Screenshot_2024-08-05_Stats.png" width="45%">
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

## Architectures amd64 and amr64.
Version 1.0.1 is operating on Linux Ubuntu 24.04 and Ubuntu Core 24 on Raspberry Pi.
Ulimately, Receiver is in development to be ported to PiCore64 for the purpose of running entirely in RAM to aviod SSD corruption from power cycling or inadvertant power removal.

# Below this line is proposed for FUTURE and in development
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

