# bbai64
Some random stuff I found related to the BeagleBone AI-64 platform

# P8 P9 headers periphery mapping
[https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec)
* [I2C](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec#I2C)
* [PWM](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec#PWM)
* [SPI](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec#SPI)

# imaging.zip, dri.zip
Taken from [TI's PROCESSOR-SDK-J721E](https://www.ti.com/tool/PROCESSOR-SDK-J721E)  
```
wget https://github.com/Hypnotriod/bbai64/raw/master/imaging.zip
sudo unzip imaging.zip -d /opt/

wget https://github.com/Hypnotriod/bbai64/raw/master/dri.zip
sudo unzip dri.zip -d /usr/lib/
```

# imx219-stereo-camera-mjpeg-stream.py
BeagleBone AI-64 MJPEG stream of Waveshare IMX219-83 Stereo Camera with GStreamer example
