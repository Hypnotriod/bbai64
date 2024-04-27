from __future__ import division
from __future__ import print_function
from __future__ import absolute_import

import os
import io
import json
import glob
import re
import argparse
import pandas as pd
import tensorflow as tf
import xml.etree.ElementTree as ET
from collections import namedtuple
from PIL import Image
from object_detection.utils import dataset_util

LABELMAP_ITEM_TEMPLATE = """item {
    id: %s
    name: '%s'
}
"""

parser = argparse.ArgumentParser()
parser.add_argument("-s", "--skip", nargs="+", default=[],
                    help="Phases to skip: prepare train export", required=False)
parser.add_argument("-p", "--python", default="python",
                    help="Python executable", required=False)
args = parser.parse_args()

with open("config.json") as f:
    config = json.load(f)

if config["disable_cuda_devices"]:
    os.environ["CUDA_VISIBLE_DEVICES"] = "-1"


def generate_folders():
    if not os.path.exists(config["labels_path"]):
        os.mkdir(config["labels_path"])
    if not os.path.exists(config["model_dir"]):
        os.mkdir(config["model_dir"])


def generate_labels(path, classes):
    f = open(path + "/labels.txt", "w")
    f.write("\n".join(classes))
    f.close()

    f = open(path + "/labelmap.pbtxt", "w")
    pbtxt = ""
    for id in range(0, len(classes)):
        pbtxt = pbtxt + LABELMAP_ITEM_TEMPLATE % (id+1, classes[id])
    f.write(pbtxt)
    f.close()


def xml_to_csv(path):
    xml_list = []
    for xml_file in glob.glob(path + "/*.xml"):
        tree = ET.parse(xml_file)
        root = tree.getroot()
        for member in root.findall("object"):
            value = (
                root.find("filename").text,
                int(root.find("size").find("width").text),
                int(root.find("size").find("height").text),
                member.find("name").text,
                int(member.find("bndbox").find("xmin").text),
                int(member.find("bndbox").find("ymin").text),
                int(member.find("bndbox").find("xmax").text),
                int(member.find("bndbox").find("ymax").text)
            )
            xml_list.append(value)
    column_name = ["filename", "width", "height",
                   "class", "xmin", "ymin", "xmax", "ymax"]
    xml_df = pd.DataFrame(xml_list, columns=column_name)
    return xml_df


def split(df, group):
    data = namedtuple('data', ['filename', 'object'])
    gb = df.groupby(group)
    return [data(filename, gb.get_group(x)) for filename, x in zip(gb.groups.keys(), gb.groups)]


def create_tf_example(group, path, labels):
    with tf.io.gfile.GFile(os.path.join(path, '{}'.format(group.filename)), 'rb') as fid:
        encoded_jpg = fid.read()
    encoded_jpg_io = io.BytesIO(encoded_jpg)
    image = Image.open(encoded_jpg_io)
    width, height = image.size

    filename = group.filename.encode('utf8')
    image_format = b'jpg'
    xmins = []
    xmaxs = []
    ymins = []
    ymaxs = []
    classes_text = []
    classes = []

    for _, row in group.object.iterrows():
        xmins.append(row['xmin'] / width)
        xmaxs.append(row['xmax'] / width)
        ymins.append(row['ymin'] / height)
        ymaxs.append(row['ymax'] / height)
        classes_text.append(row['class'].encode('utf8'))
        classes.append(labels.index(row['class'])+1)

    tf_example = tf.train.Example(features=tf.train.Features(feature={
        'image/height': dataset_util.int64_feature(height),
        'image/width': dataset_util.int64_feature(width),
        'image/filename': dataset_util.bytes_feature(filename),
        'image/source_id': dataset_util.bytes_feature(filename),
        'image/encoded': dataset_util.bytes_feature(encoded_jpg),
        'image/format': dataset_util.bytes_feature(image_format),
        'image/object/bbox/xmin': dataset_util.float_list_feature(xmins),
        'image/object/bbox/xmax': dataset_util.float_list_feature(xmaxs),
        'image/object/bbox/ymin': dataset_util.float_list_feature(ymins),
        'image/object/bbox/ymax': dataset_util.float_list_feature(ymaxs),
        'image/object/class/text': dataset_util.bytes_list_feature(classes_text),
        'image/object/class/label': dataset_util.int64_list_feature(classes),
    }))
    return tf_example


def generate_tf_record(labels, image_dir, csv_input, output_path):
    writer = tf.io.TFRecordWriter(output_path)
    path = os.path.join(image_dir)
    examples = pd.read_csv(csv_input)
    grouped = split(examples, 'filename')
    for group in grouped:
        tf_example = create_tf_example(group, path, labels)
        writer.write(tf_example.SerializeToString())

    writer.close()
    output_path = os.path.join(os.getcwd(), output_path)
    print('Successfully created the TFRecords: {}'.format(output_path))


def generate_pipeline_config():
    with open(config["base_config_path"]) as f:
        base_config = f.read()

    with open(config["pipeline_config_path"], "w") as f:
        base_config = re.sub('label_map_path: ".*?"',
                             'label_map_path: "{}"'.format(config["labelmap_path"]), base_config)
        base_config = re.sub('fine_tune_checkpoint: ".*?"',
                             'fine_tune_checkpoint: "{}"'.format(config["fine_tune_checkpoint"]), base_config)
        base_config = re.sub('(input_path: ".*?)(PATH_TO_BE_CONFIGURED/train)(.*?")',
                             'input_path: "{}"'.format(config["train_record_path"]), base_config)
        base_config = re.sub('(input_path: ".*?)(PATH_TO_BE_CONFIGURED/eval)(.*?")',
                             'input_path: "{}"'.format(config["test_record_path"]), base_config)
        base_config = re.sub('num_classes: [0-9]+',
                             'num_classes: {}'.format(len(config["classes"])), base_config)
        base_config = re.sub('batch_size: [0-9]+',
                             'batch_size: {}'.format(config["batch_size"]), base_config)
        base_config = re.sub('num_steps: [0-9]+',
                             'num_steps: {}'.format(config["num_steps"]), base_config)
        base_config = re.sub('max_detections_per_class: [0-9]+',
                             'max_detections_per_class: {}'.format(config["max_detections_per_class"]), base_config)
        base_config = re.sub('max_total_detections: [0-9]+',
                             'max_total_detections: {}'.format(config["max_total_detections"]), base_config)
        base_config = re.sub('max_number_of_boxes: [0-9]+',
                             'max_number_of_boxes: {}'.format(config["max_number_of_boxes"]), base_config)
        base_config = re.sub('fine_tune_checkpoint_type: "classification"',
                             'fine_tune_checkpoint_type: "{}"'.format('detection'), base_config)
        f.write(base_config)
        print('Successfully generated pipeline.config: {}'.format(
            config["pipeline_config_path"]))


def start_training():
    if os.system("{python} models/research/object_detection/model_main_tf2.py \
        --pipeline_config_path={pipeline_config_path} \
        --model_dir={model_dir} \
        --alsologtostderr \
        --num_train_steps={num_steps} \
        --sample_1_of_n_eval_examples=1 \
        --num_eval_steps={num_eval_steps}".format(
        python=args.python,
        pipeline_config_path=config["pipeline_config_path"],
        model_dir=config["model_dir"],
        num_steps=config["num_steps"],
        num_eval_steps=config["num_eval_steps"],
    )) != 0:
        raise Exception('Model training failed...')


def generate_evaluation_data():
    if os.system("{python} models/research/object_detection/model_main_tf2.py \
        --model_dir={model_dir} \
        --pipeline_config_path={pipeline_config_path} \
        --checkpoint_dir={checkpoint_dir}".format(
        python=args.python,
        model_dir=config["model_dir"],
        pipeline_config_path=config["pipeline_config_path"],
        checkpoint_dir=config["model_dir"],
    )) != 0:
        raise Exception('Generate evaluation data failed...')


def export_saved_model():
    print("Saving TFLite-friendly model...")
    if os.system("{python} models/research/object_detection/export_tflite_graph_tf2.py \
        --trained_checkpoint_dir {model_dir} \
        --output_directory ./ \
        --pipeline_config_path {pipeline_config_path}".format(
        python=args.python,
        model_dir=config["model_dir"],
        pipeline_config_path=config["pipeline_config_path"],
    )) != 0:
        raise Exception('Export saved model failed...')

    print("Converting TFLite model...")
    converter = tf.lite.TFLiteConverter.from_saved_model("saved_model")
    converter.optimizations = [tf.lite.Optimize.DEFAULT]
    tflite_model = converter.convert()
    tf.lite.experimental.Analyzer.analyze(
        model_content=tflite_model, gpu_compatibility=True)
    print("Saving TFLite model...")
    if not os.path.exists("saved_model_tflite"):
        os.mkdir("saved_model_tflite")
    with open("saved_model_tflite/saved_model.tflite", "wb") as f:
        f.write(tflite_model)


if "prepare" not in args.skip:
    generate_folders()
    generate_labels(config["labels_path"], config["classes"])
    print("Successfully generated: labels.txt and labelmap.pbtxt")

    xml_to_csv_input = [config["train_path"],
                        config["test_path"]]
    xml_to_csv_output = [config["train_csv_path"],
                         config["test_csv_path"]]
    for i, folder in enumerate(xml_to_csv_input):
        xml_df = xml_to_csv(folder)
        xml_df.to_csv(xml_to_csv_output[i], index=None)
    print("Successfully converted xml to csv.")

    generate_tf_record(config["classes"],
                       config["train_path"],
                       config["train_csv_path"],
                       config["train_record_path"])
    generate_tf_record(config["classes"],
                       config["test_path"],
                       config["test_csv_path"],
                       config["test_record_path"])

    generate_pipeline_config()

if "train" not in args.skip:
    start_training()
    generate_evaluation_data()

if "export" not in args.skip:
    export_saved_model()
