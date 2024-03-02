
build-mjpeg-http-mux:
	cd go/ && go build -o ../bin/mjpeg-http-mux cmd/mjpeg_http_mux/main.go && rsync -cr public/ ../bin/public/

run-mjpeg-http-mux:
	cd bin && sudo ./mjpeg-http-mux

build-comb-mjpeg-http-mux:
	cd go/ && go build -o ../bin/comb-mjpeg-http-mux cmd/comb_mjpeg_http_mux/main.go && rsync -cr public/ ../bin/public/

run-comb-mjpeg-http-mux:
	cd bin && sudo ./comb-mjpeg-http-mux

build-wifi-vehicle:
	cd go/ && go build -o ../bin/wifi-vehicle cmd/wifi_vehicle_basic/main.go && rsync -cr public/ ../bin/public/

run-wifi-vehicle:
	cd bin && sudo ./wifi-vehicle

build-wifi-two-wheeled:
	cd go/ && go build -o ../bin/wifi-two-wheeled cmd/wifi_two_wheeled_basic/main.go && rsync -cr public/ ../bin/public/

run-wifi-two-wheeled:
	cd bin && sudo ./wifi-two-wheeled

build-image-recognition:
	cd go/ && go build -o ../bin/image-recognition cmd/image_recognition/main.go && rsync -cr model/ ../bin/model/

run-image-recognition:
	cd bin && sudo ./image-recognition

run-train-image-classification:
	cd python/image_classification && python train.py
