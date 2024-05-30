# Repeater-receiver
Receiver for the DATV Repeater Project

## Overview
The receiver is based around the TBS Technologies TBS6522 Quad multi-system DVB PCIe card on a Compute Module 4 (CM4) on a daughter board that exposes the PCIe channel. Unfortunately, the standard Raspberry Pi 4 does not have the PCIe bus exposed.

The source code is developed in Go, which configures each adapter frontend to listen on a given frequency and to forward a valid TS stream and frontend stats to the repeater core.

## Design Considerations
The receiver operates independently of the repeater for several reasons:
1. **Reduce Overhead**: This approach reduces overhead in the repeater design. The concept is already in use with TVHeadend and similar projects on the internet.
2. **Configurable**: It can be configured for different PCIe DVB cards to receive both DVB-S/S2 or DVB-T/T2 per frequency. DATV uses both DVB-S/S2 and DVB-T/T2 for terrestrial experimentation.
3. **Reduce Point of Failure**: The receiver is constructed to fit in a hard disk bay of the repeater ATX 4RU chassis.

## Hardware
- Raspberry Pi Compute Module 4 (CM4004032): 4GB RAM, 32GB eMMC, without WiFi. (Reduce RF at the site)
- I2C OLED Display: Used for status display of the receiver. These OLED displays are cost-effective and an improvement over standard LCDs. I2C hardware is native to the Raspberry Pi 4 OS.
  
## Picture below showing F-Type connectors removed ready for SMA and 1:5 balun board daughter board

<img src="/docs/images/TBS-6522H-noFtypes.jpg" width="25%">
  
## Software
- **Operating System**: Ubuntu Server 24.04 LTS (headless). Considering using Ubuntu Core 22 and Snaps.
- **Build Script**: Script for setting up necessary TBS drivers and applications. There are plans to build packages to simplify the installation and avoid needing a build system and dependencies.
- **Streaming Setup**: Setting up and streaming TS from TBS6522 tested with dvblast for quick and dirty validation.
- **Application**: The application runs as a daemon, configuring and wrapping around TSduck command-line utilities to listen for a valid lock signal, forward the TS to the repeater core over RTP, and report frontend stats via a separate UDP port.

