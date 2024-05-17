package gstpipeline

import "fmt"

const SENSORS_DCC_ISP_PATH string = "/opt/imaging"

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

func CsiCameraV4l2Source(index uint) string {
	var device string
	switch index {
	case 0:
		device = "/dev/video2"
	case 1:
		device = "/dev/video18"
	}
	return fmt.Sprintf(" v4l2src device=%s", device)
}

func CsiCameraConfig(index uint, sensor Sensor, width uint, height uint) string {
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
		subdev = "/dev/v4l-subdev2"
	case 1:
		subdev = "/dev/v4l-subdev5"
	}
	return fmt.Sprintf(" ! video/x-bayer, width=%d, height=%d, format=rggb ! tiovxisp sink_0::device=%s sensor-name=%s dcc-isp-file=%s/%s/dcc_viss.bin sink_0::dcc-2a-file=%s/%s/dcc_2a.bin format-msb=%d",
		width, height, subdev, sensorName, SENSORS_DCC_ISP_PATH, sensor, SENSORS_DCC_ISP_PATH, sensor, formatMsb)
}

func UsbJpegCameraV4l2Source(index uint) string {
	var device string
	switch index {
	case 0:
		device = "/dev/video2"
	case 1:
		device = "/dev/video18"
	}
	return fmt.Sprintf(" v4l2src device=%s io-mode=2", device)
}

func UsbJpegCameraConfig(width uint, height uint) string {
	return fmt.Sprintf(" ! image/jpeg, width=%d, height=%d", width, height)
}

func GlStereoMix(leftSource string, rightSource string, leftConfig string, rightConfig string) string {
	return fmt.Sprintf(
		" -ev%s name=left%s name=right glstereomix name=mix left.%s ! glupload ! mix. right.%s ! glupload ! mix. mix. ! video/x-raw'(memory:GLMemory)', multiview-mode=side-by-side ! gldownload ! queue",
		leftSource, rightSource, leftConfig, rightConfig)
}

func VideoTestSource(width uint, height uint) string {
	return fmt.Sprintf(" videotestsrc ! video/x-raw, width=%d, height=%d",
		width, height)
}

func DecodeBin() string {
	return " ! decodebin"
}

func JpegDecode() string {
	return " ! jpegdec"
}

func VideoScale(width uint, height uint) string {
	return fmt.Sprintf(" ! videoscale method=0 add-borders=false ! video/x-raw, width=%d, height=%d",
		width, height)
}

func TiOvxMultiscaler(width uint, height uint) string {
	return fmt.Sprintf(" ! tiovxmultiscaler ! video/x-raw, width=%d, height=%d",
		width, height)
}

func TiOvxMultiscalerSplit2(width1 uint, height1 uint, pipeline1 string, width2 uint, height2 uint, pipeline2 string) string {
	return fmt.Sprintf(" ! tiovxmultiscaler name=split split. ! queue ! video/x-raw, width=%d, height=%d%s split. ! queue ! video/x-raw, width=%d, height=%d%s",
		width1, height1, pipeline1, width2, height2, pipeline2)
}

func JpegEncode(quality uint) string {
	return fmt.Sprintf(" ! jpegenc quality=%d", quality)
}

func VideoBox(left uint, right uint, top uint, bottom uint) string {
	return fmt.Sprintf(" ! videobox left=%d right=%d top=%d bottom=%d", left, right, top, bottom)
}

func TiovxdlpreprocUint8NhwcRgb(mean [3]float32, scale [3]float32) string {
	return fmt.Sprintf(" ! tiovxdlpreproc data-type=uint8 mean-0=%g mean-1=%g mean-2=%g scale-0=%g scale-1=%g scale-2=%g channel-order=nhwc tensor-format=rgb out-pool-size=4 ! application/x-tensor-tiovx",
		mean[0], mean[1], mean[2], scale[0], scale[1], scale[2])
}

func TiovxdlpreprocFloat32NhwcRgb(mean [3]float32, scale [3]float32) string {
	return fmt.Sprintf(" ! tiovxdlpreproc data-type=float32 mean-0=%g mean-1=%g mean-2=%g scale-0=%g scale-1=%g scale-2=%g channel-order=nhwc tensor-format=rgb out-pool-size=4 ! application/x-tensor-tiovx",
		mean[0], mean[1], mean[2], scale[0], scale[1], scale[2])
}

type Median uint

const (
	Median5 Median = 5
	Median9 Median = 9
)

func VideoMedian(filterSize Median) string {
	return fmt.Sprintf(" ! videomedian filtersize=%d", filterSize)
}

func VideoConvertRgba() string {
	return " ! videoconvert ! video/x-raw, format=RGBA"
}

func VideoConvertRgb() string {
	return " ! videoconvert ! video/x-raw, format=RGB"
}

func VideoConvertBgr() string {
	return " ! videoconvert ! video/x-raw, format=BGR"
}

func VideoConvertRgb16() string {
	return " ! videoconvert ! video/x-raw, format=RGB16"
}

func VideoConvertYV12() string {
	return " ! videoconvert ! video/x-raw, format=YV12"
}

func VideoConvertNV12() string {
	return " ! videoconvert ! video/x-raw, format=NV12"
}

func TiOvxDlColorConvertRgb() string {
	return " ! tiovxdlcolorconvert out-pool-size=4 ! video/x-raw, format=RGB"
}

func TiOvxDlColorConvertNV12() string {
	return " ! tiovxdlcolorconvert out-pool-size=4 ! video/x-raw, format=NV12"
}

func MjpegTcpStreamLocalhost(boundary string, port uint) string {
	return fmt.Sprintf(" ! multipartmux boundary=%s ! tcpclientsink host=127.0.0.1 port=%d",
		boundary, port)
}

func TcpStreamLocalhost(port uint) string {
	return fmt.Sprintf(" ! tcpclientsink host=127.0.0.1 port=%d", port)
}
