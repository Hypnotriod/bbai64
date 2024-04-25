# origin: https://github.com/BenGreenfield825/Tensorflow-Object-Detection-with-Tensorflow-2.0/blob/master/model_training/generate_tfrecord.py

# python generate_tfrecord.py --csv_input train_data_labels.csv --image_dir train_data --output_path train.record --labels labels/labels.txt
# python generate_tfrecord.py --csv_input test_data_labels.csv --image_dir test_data --output_path train.record --labels labels/labels.txt
# python generate_tfrecord.py --csv_input validation_data_labels.csv --image_dir validation_data --output_path train.record --labels labels/labels.txt

from __future__ import division
from __future__ import print_function
from __future__ import absolute_import

import os
import io
import argparse
import pandas as pd
import tensorflow as tf
from collections import namedtuple
from PIL import Image
from object_detection.utils import dataset_util

parser = argparse.ArgumentParser()
parser.add_argument("-i", "--csv_input",
                    help="Path to the CSV input", required=True)
parser.add_argument("-o", "--output_path",
                    help="Path to output TFRecord", required=True)
parser.add_argument("-d", "--image_dir",
                    help="Path to images", required=True)
parser.add_argument("-l", "--labels",
                    help="Path to labels.txt file", required=True)
args = parser.parse_args()


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
        classes.append(labels.index(row['class']))

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


labels = open(args.labels).read().splitlines()
writer = tf.io.TFRecordWriter(args.output_path)
path = os.path.join(args.image_dir)
examples = pd.read_csv(args.csv_input)
grouped = split(examples, 'filename')
for group in grouped:
    tf_example = create_tf_example(group, path, labels)
    writer.write(tf_example.SerializeToString())

writer.close()
output_path = os.path.join(os.getcwd(), args.output_path)
print('Successfully created the TFRecords: {}'.format(output_path))
