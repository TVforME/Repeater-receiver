# Repeater-receiver
Receiver for the DATV Repeater Project

- The reciever is based on a TBS6522 TBS-Technologies Quad receiver PCIe card to recieve both DVB-S and DVB-T transmissions.
- The source code is developed in Go for configuring, executing and sending frontend stats using the proven TSduck tools/plugins. 
- A Raspberry PI Compute Model 4 CM4004032 4GB Ram with 32GB eMMC without WiFi.
- RF information is here <Link>
- I2C OLED display for status of receiver. These OLED displays are cheap as chips and improved over a standard LCD. I2C drivers are native to Raspberry Pi4 OS.

- **Software**
- Ubuntu server 24.04 LTS (headless).  (contemplating Core 22 and Snaps?)
- build.sh script for setting up needed TBS drivers and applications. (Look at building packages to simplify the installation to avoid needing the build system and dependencies)
- Setting up and streaming TS from TBS6522 or usb dongle tested with dvblast as quick and dirty validation.
- Develope GO application as a deamon to Setup and control uses TSP dvb input to listen and forward a received dvb-s/dvb-t to the repeater core over RTP and report frontend stats via UDP port.
