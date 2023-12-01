
build-mjpeg-http-mux:
	cd go/ && go build -o ../bin/mjpeg-http-mux cmd/mjpeg_http_mux/main.go && cp -r public/ ../bin/

run-mjpeg-http-mux:
	cd bin && sudo ./mjpeg-http-mux

build-comb-mjpeg-http-mux:
	cd go/ && go build -o ../bin/comb-mjpeg-http-mux cmd/comb_mjpeg_http_mux/main.go && cp -r public/ ../bin/

run-comb-mjpeg-http-mux:
	cd bin && sudo ./comb-mjpeg-http-mux
