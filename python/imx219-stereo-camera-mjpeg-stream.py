#!/usr/bin/python
# BeagleBone AI-64 MJPEG stream of Waveshare IMX219-83 Stereo Camera with GStreamer example
# Based on https://gist.github.com/misaelnieto/2409785
# Waveshare IMX219-83 CSI Stereo Camera: https://www.waveshare.com/wiki/IMX219-83_Stereo_Camera
# To add cameras overlays modify 'fdtoverlays' property of '/boot/firmware/extlinux/extlinux.conf' with:
#   fdtoverlays /overlays/BBAI64-CSI0-imx219.dtbo /overlays/BBAI64-CSI1-imx219.dtbo
# and reboot
# Get TI's IMX219 Dynamic Camera Configuration files for Image Signal Processor:
#   wget https://github.com/Hypnotriod/bbai64/raw/master/imaging.zip
#   sudo unzip imaging.zip -d /opt/
# Make this script executable with:
#   sudo chmod +x imx219-stereo-camera-mjpeg-stream.py
# Launch script with:
#   sudo ./imx219-stereo-camera-mjpeg-stream.py
# To view stereo camera stream connect with browser to http://hostname:port ex: http://192.168.7.2:1337

import os
import time
import sys
import subprocess
from queue import Queue
from threading import Thread
from socket import socket
from select import select
from wsgiref.simple_server import WSGIServer, make_server, WSGIRequestHandler
from socketserver import ThreadingMixIn

APPLICATION_WEB_PORT = 1337
CAMERA_STREAM_1_PORT = 9990
CAMERA_STREAM_2_PORT = 9991
# JPEG_QUALITY = 85 # 0 - 100
JPEG_QUALITY = 50 # 0 - 100
# CAMERA_WIDTH = 640
# CAMERA_HEIGHT = 480
CAMERA_WIDTH = 1920
CAMERA_HEIGHT = 1080
DO_RESCALE = True
RESCALE_WIDTH = 1280
RESCALE_HEIGHT = 720

SENSOR_ISP_DRIVERS_PATH = "/opt/imaging/imx219/"
SENSOR_NAME = "SENSOR_SONY_IMX219_RPI"
CHUNK_SIZE = 4096

INDEX_PAGE = """
<html>
<head>
    <title>Waveshare IMX219-83 CSI Stereo Camera testing</title>
    <style>
        body, div, img {
            outline: none;
            margin: 0;
            padding: 0;
            background-color: black;
        }
        img {
            width: 50%;
            height: auto;
        }
        #container {
            display: flex;
            justify-content: center;
            align-items: center;
            text-align: center;
            min-height: 100vh;
        }
    </style>
</head>
<body>
    <div id="container">
        <img src="/mjpeg_stream2"/>
        <img src="/mjpeg_stream1"/>
    </div>
</body>
</html>
"""
ERROR_404 = """
<html>
  <head>
    <title>404 - Not Found</title>
  </head>
  <body>
    <h1>404 - Not Found</h1>
  </body>
</html>
"""

class StereoCameraApp(object):
    stream1_queues = []
    stream2_queues = []
    is_running = True

    def __call__(self, environ, start_response):
        if environ["PATH_INFO"] == "/":
            start_response("200 OK", [
                ("Content-Type", "text/html"),
                ("Content-Length", str(len(INDEX_PAGE)))
            ])
            return iter([INDEX_PAGE.encode()])
        elif environ["PATH_INFO"] == "/mjpeg_stream1":
            return self.stream(start_response, self.stream1_queues)
        elif environ["PATH_INFO"] == "/mjpeg_stream2":
            return self.stream(start_response, self.stream2_queues)
        else:
            start_response("404 Not Found", [
                ("Content-Type", "text/html"),
                ("Content-Length", str(len(ERROR_404)))
            ])
            return iter([ERROR_404.encode()])
    
    def launch(self):
        print("Launch input stream thread camera 1")
        t1 = Thread(target=self.input_loop, args=[self.stream1_queues, CAMERA_STREAM_1_PORT])
        t1.setDaemon(True)
        t1.start()

        print("Launch input stream thread camera 2")
        t2 = Thread(target=self.input_loop, args=[self.stream2_queues, CAMERA_STREAM_2_PORT])
        t2.setDaemon(True)
        t2.start()
        
        # CSI0 Right Camera
        print("Launch camera 1")
        t3 = Thread(target=self.start_camera, args=[
           "0", "imx219 6-0010", "/dev/video2", "/dev/v4l-subdev2", CAMERA_STREAM_1_PORT])
        t3.setDaemon(True)
        t3.start()
        
        # CSI1 Left Camera
        print("Launch camera 2")
        t4 = Thread(target=self.start_camera, args=[
            "1", "imx219 4-0010", "/dev/video18", "/dev/v4l-subdev5", CAMERA_STREAM_2_PORT])
        t4.setDaemon(True)
        t4.start()

    def stop(self):
        self.is_running = False

    def stream(self, start_response, queues):
        start_response("200 OK", [("Content-type", "multipart/x-mixed-replace; boundary=--frameboundary")])
        q = Queue()
        queues.append(q)
        while self.is_running:
            try:
                yield q.get()
            except:
                break
        if q in queues:
            queues.remove(q)

    # media-ctl -d 0 --set-v4l2 '"imx219 6-0010":0[fmt:SRGGB8_1X8/1920x1080]'
    # gst-launch-1.0 v4l2src device=/dev/video2 ! video/x-bayer, width=1920, height=1080, format=rggb ! tiovxisp sink_0::device=/dev/v4l-subdev2 sensor-name=SENSOR_SONY_IMX219_RPI dcc-isp-file=/opt/imaging/imx219/dcc_viss_1920x1080.bin sink_0::dcc-2a-file=/opt/imaging/imx219/dcc_2a_1920x1080.bin format-msb=7 ! kmssink driver-name=tidss
    def start_camera(self, media_dev, dev_name, device, subdev, port):
        time.sleep(0.2)
        os.system(f"media-ctl -d {media_dev} --set-v4l2 '\"{dev_name}\":0[fmt:SRGGB8_1X8/{CAMERA_WIDTH}x{CAMERA_HEIGHT}]'")
        cmd = f"gst-launch-1.0 v4l2src device={device} ! video/x-bayer, width={CAMERA_WIDTH}, height={CAMERA_HEIGHT}, format=rggb ! tiovxisp sink_0::device={subdev} sensor-name={SENSOR_NAME} dcc-isp-file={SENSOR_ISP_DRIVERS_PATH}dcc_viss_{CAMERA_WIDTH}x{CAMERA_HEIGHT}.bin sink_0::dcc-2a-file={SENSOR_ISP_DRIVERS_PATH}dcc_2a_{CAMERA_WIDTH}x{CAMERA_HEIGHT}.bin format-msb=7"
        if DO_RESCALE:
            cmd += f" ! tiovxmultiscaler ! video/x-raw, width={RESCALE_WIDTH}, height={RESCALE_HEIGHT}"
        cmd += f" ! jpegenc quality={JPEG_QUALITY} ! multipartmux boundary=frameboundary ! tcpclientsink host=127.0.0.1 port={port}"
        for line in self.execute(cmd):
            print(line, end="")

    def input_loop(self, queues, port):
        sock = socket()
        sock.bind(("", port))
        sock.listen(1)
        while self.is_running:
            print("Waiting for input stream on port", port)
            sd, addr = sock.accept()
            print("Accepted input stream from", addr)
            data = True
            while data:
                readable = select([sd], [], [], 0.1)[0]
                for s in readable:
                    data = s.recv(CHUNK_SIZE)
                    if not data:
                        break
                    for q in queues:
                        q.put(data)
            print("Lost input stream from", addr)
    
    def execute(self, cmd):
        popen = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, universal_newlines=True)
        for stdout_line in iter(popen.stdout.readline, ""):
            yield stdout_line 
        popen.stdout.close()
        popen.wait()

class CameraWSGIServer(ThreadingMixIn, WSGIServer):
    pass

def create_server(host, port, app, server_class=CameraWSGIServer, handler_class=WSGIRequestHandler):
    return make_server(host, port, app, server_class, handler_class)

if __name__ == "__main__":
    app = StereoCameraApp()
    print("Launching camera server on port", APPLICATION_WEB_PORT)
    httpd = create_server("", APPLICATION_WEB_PORT, app)
    app.launch()
    try:
        print("Httpd serve forever")
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("Shutdown camera server ...")
        app.stop()
        time.sleep(0.2)
        httpd.shutdown()
        sys.exit(0)
