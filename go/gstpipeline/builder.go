package gstpipeline

import "fmt"

const SENSORS_DSP_PATH string = "/opt/imaging"

type Sensor string

const (
	IMX219 Sensor = "imx219"
	IMX390 Sensor = "imx390"
)

func CsiCameraSetup(sensor Sensor, index uint, width uint, height uint) string {
	var fullName string
	switch sensor {
	case IMX219:
		switch index {
		case 0:
			fullName = "imx219 6-0010"
		case 1:
			fullName = "imx219 4-0010"
		}
		// todo IMX390
	}
	return fmt.Sprintf("media-ctl -d %d --set-v4l2 '\"%s\":0[fmt:SRGGB8_1X8/%dx%d]'",
		index, fullName, width, height)
}

func GStreamerLaunch() string {
	return "gst-launch-1.0"
}

func CsiCameraSource(sensor Sensor, index uint, width uint, height uint) string {
	var device string
	var subdev string
	var sensorName string
	var formatMsb uint
	switch sensor {
	case IMX219:
		sensorName = "SENSOR_SONY_IMX219_RPI"
		formatMsb = 7
	case IMX390:
		sensorName = "IMX390-UB953_D3"
		formatMsb = 11
	}
	switch index {
	case 0:
		device = "/dev/video2"
		subdev = "/dev/v4l-subdev2"
	case 1:
		device = "/dev/video18"
		subdev = "/dev/v4l-subdev5"
	}
	return fmt.Sprintf(" v4l2src device=%s ! video/x-bayer, width=%d, height=%d, format=rggb ! tiovxisp sink_0::device=%s sensor-name=%s dcc-isp-file=%s/%s/dcc_viss.bin sink_0::dcc-2a-file=%s/%s/dcc_2a.bin format-msb=%d",
		device, width, height, subdev, sensorName, SENSORS_DSP_PATH, sensor, SENSORS_DSP_PATH, sensor, formatMsb)
}

func TestSource(width uint, height uint) string {
	return fmt.Sprintf(" videotestsrc ! video/x-raw, width=%d, height=%d",
		width, height)
}

func DecodeBin() string {
	return " ! decodebin"
}

func Rescale(width uint, height uint) string {
	return fmt.Sprintf(" ! videoscale method=0 add-borders=false ! video/x-raw, width=%d, height=%d",
		width, height)
}

func JpegEncode(quality uint) string {
	return fmt.Sprintf(" ! jpegenc quality=%d", quality)
}

func VideoConvertRgba() string {
	return " ! videoconvert ! video/x-raw, format=RGBA"
}

func JpegTcpStreamLocalhost(boundary string, port uint) string {
	return fmt.Sprintf(" ! multipartmux boundary=%s ! tcpclientsink host=127.0.0.1 port=%d",
		boundary, port)
}

func TcpStreamLocalhost(port uint) string {
	return fmt.Sprintf(" ! tcpclientsink host=127.0.0.1 port=%d", port)
}
