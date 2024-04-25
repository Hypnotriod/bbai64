import os
import json
import tensorflow as tf

with open("conf.json") as f:
    config = json.load(f)

# print("Saving model...")
# os.system("python models/research/object_detection/exporter_main_v2.py \
#     --trained_checkpoint_dir {model_dir} \
#     --output_directory ./ \
#     --pipeline_config_path {pipeline_config_path}".format(
#     model_dir=config["model_dir"],
#     pipeline_config_path=config["pipeline_config_path"],
# ))

print("Saving TFLite-friendly model...")
os.system("python models/research/object_detection/export_tflite_graph_tf2.py \
    --trained_checkpoint_dir {model_dir} \
    --output_directory ./ \
    --pipeline_config_path {pipeline_config_path}".format(
    model_dir=config["model_dir"],
    pipeline_config_path=config["pipeline_config_path"],
))


print("Converting TFLite model...")
converter = tf.lite.TFLiteConverter.from_saved_model("saved_model")
converter.optimizations = [tf.lite.Optimize.DEFAULT]
tflite_model = converter.convert()
print("Saving TFLite model...")
os.mkdir("saved_model_tflite")
with open("saved_model_tflite/saved_model.tflite", "wb") as f:
    f.write(tflite_model)
