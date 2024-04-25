import os
import json

with open("conf.json") as f:
    config = json.load(f)

if not os.path.exists(config["labels_path"]):
    os.mkdir(config["labels_path"])
if not os.path.exists(config["base_model_dir"]):
    os.mkdir(config["base_model_dir"])
if not os.path.exists(config["model_dir"]):
    os.mkdir(config["model_dir"])

os.system("python generate_labels.py")

os.system("python xml_to_csv.py -i {train_path} {test_path} {validation_path} -o {train_csv_path} {test_csv_path} {validation_csv_path}".format(
    train_path=config["train_path"],
    test_path=config["test_path"],
    validation_path=config["validation_path"],
    train_csv_path=config["train_csv_path"],
    test_csv_path=config["test_csv_path"],
    validation_csv_path=config["validation_csv_path"],
))

os.system("python generate_tfrecord.py --csv_input {train_csv_path} --image_dir {train_path} --output_path {train_record_path} --labels {labels_path}/labels.txt".format(
    train_csv_path=config["train_csv_path"],
    train_path=config["train_path"],
    train_record_path=config["train_record_path"],
    labels_path=config["labels_path"],
))

os.system("python generate_tfrecord.py --csv_input {test_csv_path} --image_dir {test_path} --output_path {test_record_path} --labels {labels_path}/labels.txt".format(
    test_csv_path=config["test_csv_path"],
    test_path=config["test_path"],
    test_record_path=config["test_record_path"],
    labels_path=config["labels_path"],
))

os.system("python generate_tfrecord.py --csv_input {validation_csv_path} --image_dir {validation_path} --output_path {validation_record_path} --labels {labels_path}/labels.txt".format(
    validation_csv_path=config["validation_csv_path"],
    validation_path=config["validation_path"],
    validation_record_path=config["validation_record_path"],
    labels_path=config["labels_path"],
))

os.system("python generate_pipeline_config.py")
