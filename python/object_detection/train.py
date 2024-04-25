import os
import json

with open("conf.json") as f:
    config = json.load(f)

os.system("python models/research/object_detection/model_main_tf2.py \
    --pipeline_config_path={pipeline_config_path} \
    --model_dir={model_dir} \
    --alsologtostderr \
    --num_train_steps={num_steps} \
    --sample_1_of_n_eval_examples=1 \
    --num_eval_steps={num_eval_steps}".format(
    pipeline_config_path=config["pipeline_config_path"],
    model_dir=config["model_dir"],
    num_steps=config["num_steps"],
    num_eval_steps=config["num_eval_steps"],
))
