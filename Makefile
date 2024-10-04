
build-gpio-blink:
	cd go/ && go build -o ../bin/gpio-blink cmd/gpio_blink/main.go

run-gpio-blink:
	cd bin && sudo ./gpio-blink

build-csi-cam-mjpeg-stream:
	cd go/ && go build -o ../bin/csi-cam-mjpeg-stream cmd/csi_cam_mjpeg_stream/main.go && rsync -cr public/ ../bin/public/

run-csi-cam-mjpeg-stream:
	cd bin && sudo ./csi-cam-mjpeg-stream

build-usb-cam-mjpeg-stream:
	cd go/ && go build -o ../bin/usb-cam-mjpeg-stream cmd/usb_cam_mjpeg_stream/main.go && rsync -cr public/ ../bin/public/

run-usb-cam-mjpeg-stream:
	cd bin && sudo ./usb-cam-mjpeg-stream

build-csi-cam-stereo-comb-mjpeg-stream:
	cd go/ && go build -o ../bin/csi-cam-stereo-comb-mjpeg-stream cmd/csi_cam_stereo_comb_mjpeg_stream/main.go && rsync -cr public/ ../bin/public/

run-csi-cam-stereo-comb-mjpeg-stream:
	cd bin && sudo ./csi-cam-stereo-comb-mjpeg-stream

build-wifi-vehicle:
	cd go/ && go build -o ../bin/wifi-vehicle cmd/wifi_vehicle_basic/main.go && rsync -cr public/ ../bin/public/

run-wifi-vehicle:
	cd bin && sudo ./wifi-vehicle

build-wifi-two-wheeled:
	cd go/ && go build -o ../bin/wifi-two-wheeled cmd/wifi_two_wheeled_basic/main.go && rsync -cr public/ ../bin/public/

run-wifi-two-wheeled:
	cd bin && sudo ./wifi-two-wheeled

build-image-classification:
	cd go/ && go build -o ../bin/image-classification cmd/image_classification/main.go && rsync -cr model/ ../bin/model/ && rsync -cr public/ ../bin/public/

run-image-classification:
	cd bin && sudo ./image-classification

build-image-classification-tflite:
	cd go/ && go build -o ../bin/image-classification-tflite cmd/image_classification_tflite/main.go && rsync -cr model/ ../bin/model/ && rsync -cr public/ ../bin/public/

run-image-classification-tflite:
	cd bin && sudo ./image-classification-tflite

train-image-classification:
	cd python/image_classification && python3 train.py

compile-image-classification:
	export SOC=am68pa && \
	export DEVICE=j7 && \
	export TIDL_TOOLS_PATH=${TIDL_TOOLS_PATH}
	export LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:${TIDL_TOOLS_PATH} && \
	cd python/osrt_tfl && \
	python3 compile.py -c classification_config.json

build-object-detection-tflite:
	cd go/ && go build -o ../bin/object-detection-tflite cmd/object_detection_tflite/main.go && rsync -cr model/ ../bin/model/ && rsync -cr public/ ../bin/public/

run-object-detection-tflite:
	cd bin && sudo ./object-detection-tflite

train-object-detection:
	cd python/object_detection && python3 train.py --python=python3

tensorboard-object-detection:
	cd python/object_detection && tensorboard --logdir training

compile-object-detection:
	export SOC=am68pa && \
	export DEVICE=j7 && \
	export TIDL_TOOLS_PATH=${TIDL_TOOLS_PATH}
	export LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:${TIDL_TOOLS_PATH} && \
	cd python/osrt_tfl && \
	python3 compile.py -c object_detection_config.json

build-semantic-segmentation-tflite:
	cd go/ && go build -o ../bin/semantic-segmentation-tflite cmd/semantic_segmentation_tflite/main.go && rsync -cr model/ ../bin/model/ && rsync -cr public/ ../bin/public/

run-semantic-segmentation-tflite:
	cd bin && sudo ./semantic-segmentation-tflite

build-edgeai-tidl-tools-docker-container:
	docker build -t edgeai-tidl-tools-08_02_00_05 -f edgeai-tidl-tools-08_02_00_05.Dockerfile .

compile-object-detection-docker:
	docker run -it -v "$(CURDIR)/python:/workspace/python" edgeai-tidl-tools-08_02_00_05 "cd /workspace/python/osrt_tfl && python3 compile.py -c object_detection_config.json && exit"

compile-image-classification-docker:
	docker run -it -v "$(CURDIR)/python:/workspace/python" edgeai-tidl-tools-08_02_00_05 "cd /workspace/python/osrt_tfl && python3 compile.py -c classification_config.json && exit"
