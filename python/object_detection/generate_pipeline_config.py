# origin: https://github.com/BenGreenfield825/Tensorflow-Object-Detection-with-Tensorflow-2.0/blob/master/model_training/Tensorflow_2_Object_Detection_Model_Training.ipynb

import re
import json

with open("conf.json") as f:
    config = json.load(f)

with open(config["base_config_path"]) as f:
    base_config = f.read()

with open(config["pipeline_config_path"], "w") as f:
    # Set labelmap path
    base_config = re.sub('label_map_path: ".*?"',
                         'label_map_path: "{}"'.format(config["labelmap_path"]), base_config)
    # Set fine_tune_checkpoint path
    base_config = re.sub('fine_tune_checkpoint: ".*?"',
                         'fine_tune_checkpoint: "{}"'.format(config["fine_tune_checkpoint"]), base_config)
    # Set train tf-record file path
    base_config = re.sub('(input_path: ".*?)(PATH_TO_BE_CONFIGURED)(.*?")',
                         'input_path: "{}"'.format(config["train_record_path"]), base_config)
    # Set test tf-record file path
    base_config = re.sub('(input_path: ".*?)(PATH_TO_BE_CONFIGURED)(.*?")',
                         'input_path: "{}"'.format(config["test_record_path"]), base_config)
    # Set number of classes.
    base_config = re.sub('num_classes: [0-9]+',
                         'num_classes: {}'.format(len(config["classes"])-1), base_config)
    # Set batch size
    base_config = re.sub('batch_size: [0-9]+',
                         'batch_size: {}'.format(config["batch_size"]), base_config)
    # Set training steps
    base_config = re.sub('num_steps: [0-9]+',
                         'num_steps: {}'.format(config["num_steps"]), base_config)
    # Set fine-tune checkpoint type to detection
    base_config = re.sub('fine_tune_checkpoint_type: "classification"',
                         'fine_tune_checkpoint_type: "{}"'.format('detection'), base_config)
    f.write(base_config)
    print('Successfully generated pipeline.config: {}'.format(config["pipeline_config_path"]))
