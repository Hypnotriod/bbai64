# origin: https://github.com/TexasInstruments/edgeai-tidl-tools/blob/master/examples/osrt_python/tfl/tflrt_delegate.py

import yaml
import json
import shutil
import os
import argparse
import tflite_runtime.interpreter as tflite
import numpy as np
from PIL import Image

parser = argparse.ArgumentParser()
parser.add_argument("-c", "--config", help="Config JSON path", required=True)
args = parser.parse_args()
os.environ["TIDL_RT_PERFSTATS"] = "1"

with open(args.config) as f:
    config = json.load(f)

tidl_tools_path = os.environ["TIDL_TOOLS_PATH"]
calibration_images = config["calibration_images"]
required_options = {
    "tidl_tools_path": tidl_tools_path,
    "artifacts_folder": config["artifacts_path"],
}
optional_options = {
    "platform": "J7",
    "version": " 7.2",
    "tensor_bits": config["tensor_bits"],
    "debug_level": 0,
    "max_num_subgraphs": 16,
    "deny_list": "",
    "accuracy_level": 1,
    "advanced_options:calibration_frames": 2,
    "advanced_options:calibration_iterations": config["calibration_iterations"],
    "advanced_options:output_feature_16bit_names_list": "",
    "advanced_options:params_16bit_names_list": "",
    "advanced_options:quantization_scale_type": 0,
    "advanced_options:high_resolution_optimization": 0,
    "advanced_options:pre_batchnorm_fold": 1,
    "ti_internal_nc_flag": 1601,
    "advanced_options:activation_clipping": 1,
    "advanced_options:weight_clipping": 1,
    "advanced_options:bias_calibration": 1,
    "advanced_options:add_data_convert_ops":  3,
    "advanced_options:channel_wise_quantization": 0,
}


def gen_param_yaml(artifacts_folder_path, config, new_height, new_width):
    resize = []
    crop = []
    resize.append(new_width)
    resize.append(new_height)
    crop.append(new_width)
    crop.append(new_height)
    dict_file = []
    layout = "NCHW"
    if config["session_name"] == "tflitert":
        layout = "NHWC"

    model_file_name = os.path.basename(config["model_path"])

    dict_file.append({
        "task_type": config["model_type"],
        "target_device": "pc",
        "session":  {
            "artifacts_folder": "",
            "model_folder": "model",
            "model_path": model_file_name,
            "session_name": config["session_name"],
        },
        "postprocess": {
            "data_layout": layout,
        },
        "preprocess": {
            "data_layout": layout,
            "mean": config["mean"],
            "scale": config["scale"],
            "resize": resize,
            "crop": crop,
        }
    })

    if (config["model_type"] == "od"):
        if (config["od_type"] == "SSD"):
            dict_file[0]["postprocess"]["formatter"] = {
                "name": "DetectionBoxSL2BoxLS",
                "src_indices": [5, 4],
            }
        elif (config["od_type"] == "HasDetectionPostProcLayer"):
            dict_file[0]["postprocess"]["formatter"] = {
                "name": "DetectionYXYX2XYXY",
                "src_indices": [1, 0, 3, 2],
            }
        dict_file[0]["postprocess"]["detection_thr"] = 0.3

    with open(os.path.join(artifacts_folder_path, "param.yaml"), "w") as file:
        yaml.dump(dict_file[0], file)

    if (config["session_name"] == "tflitert") or (config["session_name"] == "onnxrt"):
        shutil.copy(config["model_path"], os.path.join(
            artifacts_folder_path, model_file_name))


def infer_image(interpreter, image_files, config):
    input_details = interpreter.get_input_details()
    floating_model = input_details[0]['dtype'] == np.float32
    batch = input_details[0]['shape'][0]
    height = input_details[0]['shape'][1]
    width = input_details[0]['shape'][2]
    channel = input_details[0]['shape'][3]
    new_height = height  # valid height for modified resolution for given network
    new_width = width  # valid width for modified resolution for given network
    imgs = []
    # copy image data in input_data if num_batch is more than 1
    shape = [batch, new_height, new_width, channel]
    input_data = np.zeros(shape)

    for i in range(batch):
        imgs.append(Image.open(image_files[i]).convert(
            'RGB').resize((new_width, new_height), Image.LANCZOS))
        temp_input_data = np.expand_dims(imgs[i], axis=0)
        input_data[i] = temp_input_data[0]

    if floating_model:
        input_data = np.float32(input_data)
        for mean, scale, ch in zip(config['mean'], config['scale'], range(input_data.shape[3])):
            input_data[:, :, :, ch] = (
                (input_data[:, :, :, ch] - mean) * scale)
    else:
        input_data = np.uint8(input_data)
        config['mean'] = [0, 0, 0]
        config['scale'] = [1, 1, 1]

    interpreter.resize_tensor_input(input_details[0]['index'], [
                                    batch, new_height, new_width, channel])
    interpreter.allocate_tensors()
    interpreter.set_tensor(input_details[0]['index'], input_data)
    interpreter.invoke()
    return new_height, new_width


def run_model(config):
    print("\nRunning_Model : ", config["model_name"], "\n")

    # set delegate options
    delegate_options = {}
    delegate_options.update(required_options)
    delegate_options.update(optional_options)

    if config["model_type"] == "od":
        delegate_options["object_detection:meta_layers_names_list"] = config["meta_layers_names_list"] if (
            "meta_layers_names_list" in config) else ""
        delegate_options["object_detection:meta_arch_type"] = config["meta_arch_type"] if (
            "meta_arch_type" in config) else -1

    if ("object_detection:confidence_threshold" in config and "object_detection:top_k" in config):
        delegate_options["object_detection:confidence_threshold"] = config["object_detection:confidence_threshold"]
        delegate_options["object_detection:top_k"] = config["object_detection:top_k"]

    # delete the contents of this folder
    os.makedirs(delegate_options["artifacts_folder"], exist_ok=True)
    for root, dirs, files in os.walk(delegate_options["artifacts_folder"], topdown=False):
        [os.remove(os.path.join(root, f)) for f in files]
        [os.rmdir(os.path.join(root, d)) for d in dirs]

    numFrames = len(calibration_images)

    ############   set interpreter  ################################
    delegate = tflite.load_delegate(os.path.join(
        tidl_tools_path, "tidl_model_import_tflite.so"), delegate_options)
    interpreter = tflite.Interpreter(
        model_path=config["model_path"], experimental_delegates=[delegate])
    ################################################################

    # run interpreter
    for i in range(numFrames):
        start_index = i % len(calibration_images)
        input_details = interpreter.get_input_details()
        batch = input_details[0]["shape"][0]
        input_images = []
        # for batch > 1 input images will be more than one in single input tensor
        for j in range(batch):
            input_images.append(
                calibration_images[(start_index+j) % len(calibration_images)])
        new_height, new_width = infer_image(interpreter, input_images, config)

    gen_param_yaml(delegate_options["artifacts_folder"], config, int(
        new_height), int(new_width))

    print("\nCompleted_Model : ", config["model_name"], "\n")


run_model(config)
