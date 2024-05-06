# Repeater-receiver
Receiver for the DATV Repeater Project

- The reciever is based on a TBS6522 TBS-Technologies Quad receiver PCIe card to recieve both DVB-S and DVB-T transmissions.
- The source code is developed in Go using the Linux DVB5 library. DVBlast, MuMuDVB could be used however, configuring and sending frontend stats is a requirement which unfortantanly don't do what I like.
- A Raspberry PI Compute Model 4 CM4004032 4GB Ram with 32GB  is used without WiFi for the operation running on Ubuntu headless OS
- RF information is here <Link>
- Add I2C boards to monitor LNA bias Tee current consumption using a INA219 and a EMC2101 for receiver temperature using PWM control of an internal fan to keep everything cool.
- Bi-colour led to show receivers status showing lock (however opting to us a small OLED display for status) cheap and allows to show other info

- **Software**
