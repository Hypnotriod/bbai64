
build-mjpeg-http-stream:
	cd go/ && go build -o ../bin/mjpeg-http-stream cmd/mjpeg_http_stream/main.go && rsync -cr public/ ../bin/public/

run-mjpeg-http-stream:
	cd bin && sudo ./mjpeg-http-stream

build-comb-mjpeg-http-stream:
	cd go/ && go build -o ../bin/comb-mjpeg-http-stream cmd/comb_mjpeg_http_stream/main.go && rsync -cr public/ ../bin/public/

run-comb-mjpeg-http-stream:
	cd bin && sudo ./comb-mjpeg-http-stream

build-wifi-vehicle:
	cd go/ && go build -o ../bin/wifi-vehicle cmd/wifi_vehicle_basic/main.go && rsync -cr public/ ../bin/public/

run-wifi-vehicle:
	cd bin && sudo ./wifi-vehicle

build-wifi-two-wheeled:
	cd go/ && go build -o ../bin/wifi-two-wheeled cmd/wifi_two_wheeled_basic/main.go && rsync -cr public/ ../bin/public/

run-wifi-two-wheeled:
	cd bin && sudo ./wifi-two-wheeled

build-image-recognition:
	cd go/ && go build -o ../bin/image-recognition cmd/image_recognition/main.go && rsync -cr model/ ../bin/model/ && rsync -cr public/ ../bin/public/

run-image-recognition:
	cd bin && sudo ./image-recognition

build-image-recognition-tflite:
	cd go/ && go build -o ../bin/image-recognition-tflite cmd/image_recognition_tflite/main.go && rsync -cr model/ ../bin/model/ && rsync -cr public/ ../bin/public/

run-image-recognition-tflite:
	cd bin && sudo ./image-recognition-tflite

train-image-classification:
	cd python/image_classification && python3 train.py

compile-image-classification:
	export SOC=am68pa && \
	export DEVICE=j7 && \
	export TIDL_TOOLS_PATH=${TIDL_TOOLS_PATH}
	export LD_LIBRARY_PATH=${LD_LIBRARY_PATH}:${TIDL_TOOLS_PATH} && \
	cd python/osrt_tfl && \
	python3 compile.py -c classification_config.json