# bbai64
Small projects and experiments with the BeagleBone AI-64 platform (mostly written in Go).  
Based on `bbai64-emmc-flasher-debian-11.8-xfce-edgeai-arm64-2023-10-07-10gb.img.xz` image.

# Debian 11.x (Bullseye)
[arm64-debian-11-x-bullseye-monthly-snapshots-2023-10-07](https://forum.beagleboard.org/t/arm64-debian-11-x-bullseye-monthly-snapshots-2023-10-07/32318)

# P8 P9 headers periphery mapping
[https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec)
* [I2C](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec#I2C)
* [PWM](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec#PWM)
* [SPI](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec#SPI)

# Go installation  
[Latest Go toolchain builds](https://go.dev/dl/) 
```
wget https://go.dev/dl/go1.21.6.linux-arm64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.6.linux-arm64.tar.gz
```
Update `~/.bashrc` with
```
export PATH=$PATH:/usr/local/go/bin
```
Apply changes with
```
source ~/.bashrc
```

# imaging.zip, dri.zip
Taken from [TI's PROCESSOR-SDK-J721E](https://www.ti.com/tool/PROCESSOR-SDK-J721E)  
```
wget https://github.com/Hypnotriod/bbai64/raw/master/imaging.zip
sudo unzip imaging.zip -d /opt/

wget https://github.com/Hypnotriod/bbai64/raw/master/dri.zip
sudo unzip dri.zip -d /usr/lib/
```

# libtensorflowlite_c 2.9.0 for linux arm64
```
wget https://github.com/Hypnotriod/bbai64/raw/master/libtensorflowlite_c-2.9.0-linux-arm64.tar.gz
sudo tar -C /usr/local -xvf libtensorflowlite_c-2.9.0-linux-arm64.tar.gz
sudo ldconfig
```

# libtensorflow 2.4.1 for linux arm64
```
wget https://github.com/kesuskim/libtensorflow-2.4.1-linux-arm64/raw/master/libtensorflow.tar.gz
sudo tar -C /usr/local -xvf libtensorflow.tar.gz
sudo ldconfig
```

# Prepare edgeai-tidl-tools on Ubuntu PC
* Clone [edgeai-tidl-tools](https://github.com/TexasInstruments/edgeai-tidl-tools)
```
sudo apt-get install libyaml-cpp-dev
sudo apt-get install cmake
cd edgeai-tidl-tools
git checkout 08_02_00_05 -b 08_02_00_05
export SOC=am68pa
./setup.sh
```

# Compile tflite model artifacts for tidl delegate on Ubuntu PC
```
make compile-image-classification TIDL_TOOLS_PATH=/path_to_tidl_tools/edgeai-tidl-tools/tidl_tools/
```

# Object detection
[protocolbuffers_v3.20](https://github.com/protocolbuffers/protobuf/releases/tag/v3.20.3)
* Download and extract model content from [tf2_detection_zoo](https://github.com/tensorflow/models/blob/master/research/object_detection/g3doc/tf2_detection_zoo.md) to `python/object_detection/base_model` folder
* Update in `python/object_detection/base_model/pipeline.config` the `input_path: "PATH_TO_BE_CONFIGURED"` fields of `train_input_reader` and `eval_input_reader` with `"PATH_TO_BE_CONFIGURED/train"` and `"PATH_TO_BE_CONFIGURED/eval"` respectively
```
cd python/object_detection
conda create --name tensorflow_od
conda activate tensorflow_od
conda install python=3.7
pip install -r requirements.txt
git clone --depth 1 https://github.com/tensorflow/models.git
cd models/research/
protoc object_detection/protos/*.proto --python_out=.
cp object_detection/packages/tf2/setup.py .
python3 -m pip install .
pip install protobuf==3.20.3
```
* `train.py` - to train the model. 
  * `--skip` - to skip phases `prepare` `train` `export`

# wifi vehicle hardware
* 2 channels RC car platform with steering servo and ESC (Electronic Speed Control)
* 3.3v to 5-6v PWM signal conversion circuit
* Arducam IMX219 sensor based Camera Module with 15-pin to 22-pin FPC (Flexible Printed Circuit) cable
* Waveshare UPS Module 3S for BBAI64 powering and power monitoring
* Gamepad for use as the car controller on the web page

# Build and run go apps with make example
```
make build-wifi-vehicle
make run-wifi-vehicle
```

# imx219-stereo-camera-mjpeg-stream.py
BeagleBone AI-64 MJPEG stream of Waveshare IMX219-83 Stereo Camera with GStreamer example
