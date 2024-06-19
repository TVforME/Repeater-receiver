# Repeater-receiver
Receiver for the DATV Repeater Project

## Overview
The receiver is based around the TBS Technologies TBS6522 Quad multi-system DVB PCIe card on a Compute Module 4 (CM4) on a daughter board that exposes the PCIe channel. Unfortunately, the standard Raspberry Pi 4 by default does not have the PCIe bus exposed. USB dirived DVB sticks are fine with a simple use case however, limitited to how many usb ports available. PCI connection can support 8 individual adaptors at once over PCIe x 1 expander board.

The source code (prototype) was developed in pure bash script however, likely to be converted to GO to remove the need for dvblast and socat.

DVBlast and DVBlastctl are configured in combination to listen for a valid TS and signal lock and stream a single TS service via RTP to endpoints. 

The script configures each adapter frontend's frequency and other necessary parameters, creates individual stream config files then spins up dvblast for each adaptor. frontend stats are reported each (x) seconds via UDP to the repeater core. However, 

I have modified the stats reporting from any adaptor frontend to issue a lock only to the repeater core where the stats generated locally using lighthttpd and overlayed using wpe on the core? There is certainly, enough resources to spin-up a http server to host statistics.

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
- **Aplication**: Entirely bash script (prototype for development) move to => deamon in future coded in Go.
- **Build Script**: Script for setting up necessary v4ldvb, TBS drivers, a few tools such as pcitools and the dvblast application. I'll develop a tcz package to load on sartup with all dependancies included. DVBlast leverages of BiTstream and libev libraries.
- **Streaming Setup**: dvb.conf is read to configure adaptors and begin streaming.  signal lock is Multicast over UDP using socat. I  used ncat initially however, it didn't cut it.

