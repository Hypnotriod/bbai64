#!/usr/bin/python
# BeagleBone AI-64 streaming Waveshare IMX219-83 Stereo Camera MJPEG with GStreamer example
# Based on https://gist.github.com/misaelnieto/2409785
# Waveshare IMX219-83 CSI Stereo Camera: https://www.waveshare.com/wiki/IMX219-83_Stereo_Camera
# To add cameras overlays modyfy 'fdtoverlays' property of '/boot/firmware/extlinux/extlinux.conf' with:
# fdtoverlays /overlays/BBAI64-CSI0-imx219.dtbo /overlays/BBAI64-CSI1-imx219.dtbo
# Get TI Drivers:
#   wget https://github.com/Hypnotriod/bbai64/raw/master/imaging.zip
#   sudo unzip imaging.zip -d opt/
# Launch script with sudo ./imx219-stereo-camera-mjpeg-stream.py
# To view stereo camera stream connect with browser to http://{hostname}:{appport} ex: http://192.168.7.2:1337

import os
import time
import sys
from queue import Queue
from threading import Thread
from socket import socket
from select import select
from wsgiref.simple_server import WSGIServer, make_server, WSGIRequestHandler
from socketserver import ThreadingMixIn

APPLICATION_WEB_PORT = 1337
CAMERA_STREAM_1_PORT = 9990
CAMERA_STREAM_2_PORT = 9991
CAMERA_WIDTH = 640
CAMERA_HEIGHT = 480
SENSOR_ISP_DRIVERS_PATH = "/opt/imaging/imx219/"
SENSOR_NAME = "SENSOR_SONY_IMX219_RPI"

class CameraWSGIServer(ThreadingMixIn, WSGIServer):
    pass

def create_server(host, port, app, server_class=CameraWSGIServer,
        handler_class=WSGIRequestHandler):
    return make_server(host, port, app, server_class, handler_class)

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
            position: relative;
            top: 50%;
            transform: translateY(-50%);
        }
    </style>
</head>
<body>
    <div id="container">
        <img src="/mjpeg_stream1"/>
        <img src="/mjpeg_stream2"/>
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


class IPCameraApp(object):
    stream1Queues = []
    stream2Queues = []

    def __call__(self, environ, start_response):
        if environ['PATH_INFO'] == '/':
            start_response("200 OK", [
                ("Content-Type", "text/html"),
                ("Content-Length", str(len(INDEX_PAGE)))
            ])
            return iter([INDEX_PAGE.encode()])
        elif environ['PATH_INFO'] == '/mjpeg_stream1':
            return self.stream(start_response, self.stream1Queues)
        elif environ['PATH_INFO'] == '/mjpeg_stream2':
            return self.stream(start_response, self.stream2Queues)
        else:
            start_response("404 Not Found", [
                ("Content-Type", "text/html"),
                ("Content-Length", str(len(ERROR_404)))
            ])
            return iter([ERROR_404.encode()])

    def stream(self, start_response, queues):
        start_response('200 OK', [('Content-type', 'multipart/x-mixed-replace; boundary=--spionisto')])
        q = Queue()
        queues.append(q)
        while True:
            try:
                yield q.get()
            except:
                if q in queues:
                    queues.remove(q)
                return


def input_loop(queues, port):
    sock = socket()
    sock.bind(('', port))
    sock.listen(1)
    while True:
        print('Waiting for input stream on port', port)
        sd, addr = sock.accept()
        print('Accepted input stream from', addr)
        data = True
        while data:
            readable = select([sd], [], [], 0.1)[0]
            for s in readable:
                data = s.recv(1024)
                if not data:
                    break
                for q in queues:
                    q.put(data)
        print('Lost input stream from', addr)

def start_camera1(width, height, port):
    time.sleep(1)
    os.system(f'media-ctl -d 0 --set-v4l2 \'"imx219 6-0010":0[fmt:SRGGB8_1X8/{width}x{height}]\'')
    os.system(f'gst-launch-1.0 v4l2src device=/dev/video2 ! video/x-bayer, width={width}, height={height}, format=rggb ! tiovxisp sink_0::device=/dev/v4l-subdev2 sensor-name={SENSOR_NAME} dcc-isp-file={SENSOR_ISP_DRIVERS_PATH}dcc_viss_{width}x{height}.bin sink_0::dcc-2a-file={SENSOR_ISP_DRIVERS_PATH}dcc_2a_{width}x{height}.bin format-msb=7 ! jpegenc ! multipartmux boundary=spionisto ! tcpclientsink host=127.0.0.1 port={port}')

def start_camera2(width, height, port):
    time.sleep(1)
    os.system(f'media-ctl -d 1 --set-v4l2 \'"imx219 4-0010":0[fmt:SRGGB8_1X8/{width}x{height}]\'')
    os.system(f'sudo gst-launch-1.0 v4l2src device=/dev/video18 ! video/x-bayer, width={width}, height={height}, format=rggb ! tiovxisp sink_0::device=/dev/v4l-subdev5 sensor-name={SENSOR_NAME} dcc-isp-file={SENSOR_ISP_DRIVERS_PATH}dcc_viss_{width}x{height}.bin sink_0::dcc-2a-file={SENSOR_ISP_DRIVERS_PATH}dcc_2a_{width}x{height}.bin format-msb=7 ! jpegenc ! multipartmux boundary=spionisto ! tcpclientsink host=127.0.0.1 port={port}')

if __name__ == '__main__':

    #Launch an instance of wsgi server
    app = IPCameraApp()
    print('Launching camera server on port', APPLICATION_WEB_PORT)
    httpd = create_server('', APPLICATION_WEB_PORT, app)

    print('Launch input stream thread camera 1')
    t1 = Thread(target=input_loop, args=[app.stream1Queues, CAMERA_STREAM_1_PORT])
    t1.setDaemon(True)
    t1.start()

    print('Launch input stream thread camera 2')
    t2 = Thread(target=input_loop, args=[app.stream2Queues, CAMERA_STREAM_2_PORT])
    t2.setDaemon(True)
    t2.start()
    
    print('Launch camera 1')
    t3 = Thread(target=start_camera1, args=[CAMERA_WIDTH, CAMERA_HEIGHT, CAMERA_STREAM_1_PORT])
    t3.setDaemon(True)
    t3.start()
    
    print('Launch camera 2')
    t4 = Thread(target=start_camera2, args=[CAMERA_WIDTH, CAMERA_HEIGHT, CAMERA_STREAM_2_PORT])
    t4.setDaemon(True)
    t4.start()

    try:
        print('Httpd serve forever')
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("Shutdown camera server ...")
        httpd.shutdown()
        sys.exit()
        