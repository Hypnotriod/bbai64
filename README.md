# bbai64
Small projects and experiments with the BeagleBone AI-64 platform written in Go and Python.  
Includes custom `Image Classification` and `Object Detection` train scripts with Tensorflow as well as scripts for further models compilation for `TI TFlite Delegate`.  
Based on 
[BBAI64 11.8 2023-10-07 10GB eMMC TI EDGEAI Xfce Flasher](https://www.beagleboard.org/distros/bbai64-11-8-2023-10-07-10gb-emmc-ti-edgeai-xfce-flasher) image.  
`setup_script.sh` from `/opt/edge_ai_apps/` must be run to install `edgeai-gst-plugins` 
```bash
cd /opt/edge_ai_apps/ && sudo ./setup_script.sh
```

# Overlays
To add support of various periphery as well as IMX219 CSI cameras `fdtoverlays` property of `/boot/firmware/extlinux/extlinux.conf` should be modified with `/overlay/THE_OVERLAY_NAME.dtbo`. For example:
```txt
fdtoverlays /overlays/BONE-PWM0.dtbo /overlays/BONE-PWM1.dtbo /overlays/BONE-I2C1.dtbo /overlays/BBAI64-CSI0-imx219.dtbo /overlays/BBAI64-CSI1-imx219.dtbo
```
Checkout [arm64 overlays list](https://git.beagleboard.org/beagleboard/BeagleBoard-DeviceTrees/-/tree/v5.10.x-ti-unified/src/arm64/overlays)

# P8 P9 headers periphery mapping
[https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec)
* [I2C](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec#I2C)
* [PWM](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec#PWM)
* [SPI](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec#SPI)
* [UART](https://elinux.org/Beagleboard:BeagleBone_cape_interface_spec#UART)
* [GPIO](
https://github.com/Hypnotriod/bbai64/blob/master/GPIO.md)

# Docs
[edgeai_dataflows](https://software-dl.ti.com/jacinto7/esd/processor-sdk-linux-edgeai/TDA4VM/08_06_01/exports/docs/common/edgeai_dataflows.html)  
[gstreamer plugins](https://gstreamer.freedesktop.org/documentation/plugins_doc.html?gi-language=c)  

# Go installation  
[Latest Go toolchain builds](https://go.dev/dl/) 
```shell
wget https://go.dev/dl/go1.21.6.linux-arm64.tar.gz
sudo rm -rf /usr/local/go
sudo rm -rf /usr/bin/go
sudo tar -C /usr/local -xzf go1.21.6.linux-arm64.tar.gz
```
Update `~/.bashrc` with
```shell
export PATH=$PATH:/usr/local/go/bin
```
Apply changes and check go version
```shell
source ~/.bashrc
go version
```

# IMX219 Dynamic Camera Configuration files for Image Signal Processor
Taken from [TI's PROCESSOR-SDK-J721E](https://www.ti.com/tool/PROCESSOR-SDK-J721E)  
Required by `tiovxisp` **gstreamer** plugin to work with IMX219 SCI camera.
```shell
wget https://github.com/Hypnotriod/bbai64/raw/master/imaging.zip
sudo unzip imaging.zip -d /opt/
```

# libtensorflowlite_c.so 2.9.0 for linux arm64
```shell
wget https://github.com/Hypnotriod/bbai64/raw/master/libtensorflowlite_c-2.9.0-linux-arm64.tar.gz
sudo tar -C /usr/local -xvf libtensorflowlite_c-2.9.0-linux-arm64.tar.gz
sudo ldconfig
```

# libtensorflow.2.4.1.so for linux arm64
```shell
wget https://github.com/kesuskim/libtensorflow-2.4.1-linux-arm64/raw/master/libtensorflow.tar.gz
sudo tar -C /usr/local -xvf libtensorflow.tar.gz
sudo ldconfig
```

# Prepare edgeai-tidl-tools on Ubuntu PC
```shell
sudo apt-get install libyaml-cpp-dev
sudo apt-get install cmake
conda create --name tensorflow_tidl
conda activate tensorflow_tidl
conda install python=3.7
git clone --depth 1 --branch 08_02_00_05 https://github.com/TexasInstruments/edgeai-tidl-tools
cd edgeai-tidl-tools
export SOC=am68pa
./setup.sh
```

# Compile tflite model artifacts for tidl delegate on Ubuntu PC
[TI Deep Learning Library User Guide](https://software-dl.ti.com/jacinto7/esd/processor-sdk-rtos-jacinto7/07_03_00_07/exports/docs/tidl_j7_02_00_00_07/ti_dl/docs/user_guide_html/md_tidl_osr_tflrt_tidl.html)  
[User options for TIDL Acceleration](https://github.com/TexasInstruments/edgeai-tidl-tools/blob/master/examples/osrt_python/README.md)
```shell
make compile-image-classification TIDL_TOOLS_PATH=/path_to_tidl_tools/edgeai-tidl-tools/tidl_tools/
make compile-object-detection TIDL_TOOLS_PATH=/path_to_tidl_tools/edgeai-tidl-tools/tidl_tools/
```

# Compile tflite model artifacts for tidl delegate using Docker container
```shell
make build-edgeai-tidl-tools-docker-container
make compile-object-detection-docker
make compile-image-classification-docker
```

# CUDA, cuDNN
[tensorflow cuDNN CUDA configuration list](https://www.tensorflow.org/install/source#gpu)  
[cuda-11.2.0 for tensorflow 2.10.1](https://developer.nvidia.com/cuda-11.2.0-download-archive)  
[cuda-10.0 for tensorflow 1.15](https://developer.nvidia.com/cuda-10.0-download-archive)  
[cudnn-archive](https://developer.nvidia.com/rdp/cudnn-archive)  

# Image classification custom model training on Ubuntu / Windows PC
* Prepare [conda](https://conda.io/projects/conda/en/latest/user-guide/install/index.html) environment
```shell
cd python/image_classification
conda create --name tensorflow_ic
conda activate tensorflow_ic
conda install python=3.7
pip install -r requirements.txt
```
* Add your `train` *(to train and validate)* and `test` *(to test the final result)* images to `train_data` and `test_data` folders respectively. Each image related to specific `class` should be in its own subfolder *named by the class name*.
* `config.json` - training configuration file.
  * Fill `classes` field with your class names in desired order.
  * You may tweak the `epochs`, `validation_split`, `batch_size` e.t.c
* `train.py` - script to train the model.
  * At the end of successful training should generate `labels/labels.txt` and `saved_model_tflite/saved_model.tflite` files.

# Object detection custom model training on Ubuntu / Windows PC
[protocolbuffers_v3.20](https://github.com/protocolbuffers/protobuf/releases/tag/v3.20.3) 
* Prepare [conda](https://conda.io/projects/conda/en/latest/user-guide/install/index.html) environment
```shell
cd python/object_detection
conda create --name tensorflow_od
conda activate tensorflow_od
conda install python=3.7
pip install -r requirements_tf1.txt
# pip install -r requirements_tf2.txt
git clone --depth 1 https://github.com/tensorflow/models.git
# git clone https://github.com/tensorflow/models.git && git reset --hard a0d092533701cbbf4cde97337b1e4aac51943c4d
cd models/research/
protoc object_detection/protos/*.proto --python_out=.
cp object_detection/packages/tf1/setup.py .
# cp object_detection/packages/tf2/setup.py .
python3 -m pip install .
pip install protobuf==3.20.3
```
* Prepare you images annotation with [labelImg](https://github.com/HumanSignal/labelImg) graphical image annotation tool. Should generate annotation `xml` file for each image file.
* Add your `train` *(for training)* and `test` *(for evaluation)* images and xmls files to `train_data` and `test_data` folders respectively. 
* Download and extract the content of your model of choise from link below and put into `python/object_detection/base_model` folder, for example: [ssd_mobilenet_v2_coco](http://download.tensorflow.org/models/object_detection/ssd_mobilenet_v2_coco_2018_03_29.tar.gz)
  * [tf1_detection_zoo](https://github.com/tensorflow/models/blob/master/research/object_detection/g3doc/tf1_detection_zoo.md) 
  * [tf2_detection_zoo](https://github.com/tensorflow/models/blob/master/research/object_detection/g3doc/tf2_detection_zoo.md)
* Update in `python/object_detection/base_model/pipeline.config` the `input_path: "PATH_TO_BE_CONFIGURED"` fields of `train_input_reader` and `eval_input_reader` with `"PATH_TO_BE_CONFIGURED/train"` and `"PATH_TO_BE_CONFIGURED/eval"` respectively
* Also delete if exists line `batch_norm_trainable: true` from `python/object_detection/base_model/pipeline.config`
* `config.json` - training configuration file.
  * Check the `base_config_path` and `fine_tune_checkpoint` paths to your model. 
  * Update the `input_shapes` with the input shape of your model. Check `fixed_shape_resizer` field in `pipeline.config` 
  * Fill `classes` field with your class names in desired order.
  * You may tweak the `num_steps`, `batch_size` e.t.c
* `train.py` - script to train the model.
  * `--skip` - to skip phases `prepare` `train` `export`
  * At the end of successful training should generate `labels/labels.txt` and `saved_model_tflite/saved_model.tflite` files.

# wifi vehicle hardware
* 2 channels RC car platform with steering servo and ESC (Electronic Speed Control)
* 3.3v to 5-6v PWM signal conversion circuit
* Arducam IMX219 sensor based Camera Module with 15-pin to 22-pin FPC (Flexible Printed Circuit) cable
* Waveshare UPS Module 3S for BBAI64 powering and power monitoring
* Gamepad for use as the car controller on the web page

# Build and run go apps with make example
```shell
make build-wifi-vehicle
make run-wifi-vehicle
```

# imx219-stereo-camera-mjpeg-stream.py
BeagleBone AI-64 MJPEG stream of Waveshare IMX219-83 Stereo Camera with GStreamer example
