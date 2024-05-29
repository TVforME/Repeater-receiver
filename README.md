# Repeater-receiver
Receiver for the DATV Repeater Project

- The reciever is based on a TBS6522 TBS-Technologies Quad receiver PCIe card to recieve both DVB-S and DVB-T transmissions.
- The source code is developed in Go using the Linux DVB5 library. DVBlast, MuMuDVB could be used however, configuring and sending frontend stats is a requirement which unfortantanly don't do what I like.
- A Raspberry PI Compute Model 4 CM4004032 4GB Ram with 32GB  is used without WiFi for the operation running on Ubuntu headless OS
- RF information is here <Link>
- Add I2C OLED display for status of receiver. Cheap and allows to show other info over the first idea of a flashing bi-clor LED.

- **Software**
