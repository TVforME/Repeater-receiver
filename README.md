# Repeater-receiver
Receiver for the DATV Repeater Project

## Overview
The receiver is based around the TBS Technologies TBS6522 Quad multi-system DVB PCIe card on a Compute Module 4 (CM4) on a daughter board that exposes the PCIe channel. Unfortunately, the standard Raspberry Pi 4 does not have the PCIe bus exposed to allow connection with PCI hardware. USB dirived DVB stcks are fine with raspberry pi 4 however, limitited to how many usb ports available. PCI connection can support 8 individual adaptors at once.

The source code (prototype) has been developed in pure bash script however, likely to be converted to GO. 

DVBlast and DVBlastctl are used to listen and stream TS via RTP to endpoints. The script configures each adapter frontend to listen on a given frequency and to forward a valid TS multicast stream. frontend stats are reported each (x) seconds via UDP to the repeater core. However, contemptating simplying to reporting adaptor frontend lock only to the repeater core with the stats generated locally using lighthttpd and overlayed using wpe on the core?

# Using piCore64 on a Raspberry Pi Compute Module 4 (CM4) 
piCore offers significant advantages for systems where reliability and SD card/USB/EMMC lifespan are critical.  Why piCore64?

piCore64 is a variant of [Tiny Core Linux](http://tinycorelinux.net/) is designed specifically for 64-bit systems. It's extremely lightweight and operates predominantly in RAM. This means that after the initial boot process, where the system image is loaded into memory, piCore64 runs entirely from RAM from there on. Benefits of Running Linux from RAM are:

1. Reduction of Write Operations:
Traditional Linux distributions write data frequently to the storage medium (like SSD drives, SD cards, USB sticks, or EMMC). Each write operation potentially shortens the lifespan of these storage devices due to wear and tear on the flash cells. piCore64 minimizes this risk by operating in RAM, thus significantly reducing the number of write operations to flash device is virtuallt zero.

2. Increased System Performance:
RAM is significantly faster than most forms of persistent storage, particularly SD cards and EMMC. By operating from RAM, piCore64 ensures that the system can run applications and processes much faster, which is crucial for real-time applications like such as mission critical applications.

4. Enhanced Reliability and Stability:
Since piCore64 minimises disk/SD/eMMC writes, the risk of file system corruption due to unexpected power failures or write failures is greatly reduced. This is particularly important for systems that may be deployed in remote or less accessible locations, where maintenance can be challenging.

4. Simplified System Maintenance:
With fewer writes to the storage device, the overall system maintenance is reduced. There's less need for regular file system checks and reduced concern about data integrity issues once to power goes off at a repeater site.

Developing the repeater receiver code on piCore64 allows the Raspberry Pi CM4 to perform efficiently and reliably in a broadcast environment. The operating system's lightweight ensures that most of the Raspberry Pi’s resources are dedicated to handling the communications with dvb adaptors and multiple instances of DVBlast rather than system overheads such as Snap/apt triggers and many deamons hiding in the background. Additionally, the ephemeral nature of RAM-based systems means that any configuration changes or temporary data are reset upon reboot, which can help in maintaining a consistent state across power cycles.
Despite, loading settings into RAM on startup, piCore has facilities to make settings persistant by running simple scripts. 

## Design Considerations
The receiver operates independently of the repeater for several reasons:
1. **Reduce Overhead**: This approach reduces overhead in the repeater Core design. The concept was to keep all the RF in one box and interconnect using Ethernet RTP/UDP and HTTP.  Technaically, the receive could be located remotely from the core.
2. **Configurable**: It can be configured for different PCIe DVB cards to receive both DVB-S/S2 or DVB-T/T2 per frequency. DATV uses both DVB-S/S2 and DVB-T/T2 for terrestrial experimentation.
3. **Reduce Point of Failure**: The receiver is constructed to fit in a hard disk bay of the repeater ATX 4RU chassis. Duplicate receive is to be build to facilitate "upgrades and features" and simply replace the existing receiver in 5 minutes making servicability key importance at repeater sites.

## Hardware
- Raspberry Pi Compute Module 4 (CM4004032): 4GB RAM, 32GB eMMC, without WiFi. (Reduce RF at the site) No need to access WiFi 
- I2C OLED Display: Used for status display of the receiver. These OLED displays are cost-effective and an improvement over standard LCDs. I2C software is native in later Raspberrian including fan control and RTC hardware.
  
## Picture below showing F-Type connectors removed ready for SMA and 1:5 balun board daughter board

<img src="/docs/images/TBS-6522H-noFtypes.jpg" width="25%">
  
## Software
- **Operating System**: piCore64 version 14.1.0 
- **Build Script**: Script for setting up necessary v4l dvb, TBS drivers, pci tools and dvblast application. I'll develope a tcz package to load for anumber of dependencies.
- **Streaming Setup**: dvb.conf is read to configure adaptors and begin streaming on signal lock.
- **Application**: The final application runs as a daemon.. 

