
build-mjpeg-http-mux:
	cd go/ && go build -o ../bin/mjpeg-http-mux cmd/mjpeg_http_mux/main.go && cp -r public/ ../bin/

run-mjpeg-http-mux:
	cd bin && sudo ./mjpeg-http-mux

build-rgba-stereo-to-mjpeg-http-mux:
	cd go/ && go build -o ../bin/rgba-stereo-to-mjpeg-http-mux cmd/rgba_stereo_to_mjpeg_http_mux/main.go && cp -r public/ ../bin/

run-rgba-stereo-to-mjpeg-http-mux:
	cd bin && sudo ./rgba-stereo-to-mjpeg-http-mux
