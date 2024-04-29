# origin: https://github.com/amineHorseman/mobilenet-v2-custom-dataset

# img_size: 224, 192, 160, 128, 96 (https://github.com/JonathanCMitchell/mobilenet_v2_keras/blob/master/mobilenetv2.py)

import json
import datetime
import time
import glob
import numpy as np
import os
import warnings
import tensorflow as tf
from tensorflow import keras
from keras.applications.mobilenet_v2 import MobileNetV2
from keras.models import Model
from keras.layers import Dense, Input
from keras.utils import img_to_array, load_img, image_dataset_from_directory
from keras.optimizers import SGD
from keras.callbacks import ModelCheckpoint
from keras.applications.mobilenet_v2 import preprocess_input

warnings.simplefilter(action="ignore", category=FutureWarning)
os.environ["TF_CPP_MIN_LOG_LEVEL"] = "3"
os.environ["TF_DISABLE_RZ_CHECK"] = "1"

with open("config.json") as f:
    config = json.load(f)

disable_cuda_devices = config["disable_cuda_devices"]
weights = config["weights"]
img_size = config["img_size"]
learning_rate = config["learning_rate"]
momentum = config["momentum"]
train_path = config["train_path"]
test_path = config["test_path"]
model_path = config["model_path"]
tflite_model_path = config["tflite_model_path"]
labels_path = config["labels_path"]
batch_size = config["batch_size"]
epochs = config["epochs"]
classes = config["classes"]
shuffle = config["shuffle"]
seed = config["seed"]
validation_split = config["validation_split"]
checkpoint_monitor = config["checkpoint_monitor"]
epochs_after_unfreeze = config["epochs_after_unfreeze"]
checkpoint_period = config["checkpoint_period"]
checkpoint_period_after_unfreeze = config["checkpoint_period_after_unfreeze"]

if disable_cuda_devices:
    os.environ["CUDA_VISIBLE_DEVICES"] = "-1"


def write_labels(path):
    f = open(path + "/labels.txt", "w")
    f.write("\n".join(classes))
    f.close()


def create_folders():
    if not os.path.exists(model_path):
        os.mkdir(model_path)
    if tflite_model_path and not os.path.exists(tflite_model_path):
        os.mkdir(tflite_model_path)
    if not os.path.exists(labels_path):
        os.mkdir(labels_path)
    if not os.path.exists("logs"):
        os.mkdir("logs")


def preprocess(images, labels):
    return preprocess_input(images), labels


def train(checkpoint, epochs):
    train_data, validation_data = image_dataset_from_directory(
        train_path,
        validation_split=validation_split,
        labels="inferred",
        label_mode="categorical",
        class_names=classes,
        subset="both",
        seed=seed,
        shuffle=shuffle,
        batch_size=batch_size,
        image_size=(img_size, img_size)
    )
    model.fit(
        train_data.map(preprocess),
        epochs=epochs,
        verbose=1,
        validation_data=validation_data.map(preprocess),
        callbacks=[checkpoint])


def test():
    folders = [name for name in os.listdir(
        test_path) if os.path.isdir(os.path.join(test_path, name))]
    for folder in folders:
        success = 0
        average_confidence = 0
        files = glob.glob(test_path + "/" + folder + "/*")
        for file in files:
            img = load_img(file, target_size=(img_size, img_size))
            x = img_to_array(img)
            x = np.expand_dims(x, axis=0)
            x = preprocess_input(x)
            y_prob = model.predict(x, verbose=0)
            y_class = y_prob.argmax(axis=-1)
            y_class = y_class[0]
            y_confidence = int(y_prob[0][y_class] * 100)
            if y_class == classes.index(folder):
                success += 1
            average_confidence += y_confidence
        success = int(100*success/len(files))
        average_confidence = int(average_confidence / len(files))
        print("class '{}': success rate = {}% with {}% avg confidence".format(
            folder, success, average_confidence))


create_folders()

# create model
base_model = MobileNetV2(include_top=True, weights=weights,
                         input_tensor=Input(shape=(img_size, img_size, 3)), input_shape=(img_size, img_size, 3))
predictions = Dense(len(classes), activation="softmax")(
    base_model.layers[-2].output)
model = Model(inputs=base_model.input, outputs=predictions)

print("[INFO] Model is successfully loaded and patched...")
model.summary()

# create callbacks
checkpoint = ModelCheckpoint(
    "logs/weights.h5", monitor=checkpoint_monitor, save_best_only=True, save_freq=checkpoint_period)

# start time
start = time.time()

print("Freezing the base layers. Unfreeze the last layer...")
for layer in model.layers[:-1]:
    layer.trainable = False
model.compile(optimizer="rmsprop", loss="categorical_crossentropy")

print("Start training...")
train(checkpoint, epochs)

if epochs_after_unfreeze > 0:
    print("Unfreezing all layers...")
    for i in range(len(model.layers)):
        model.layers[i].trainable = True
    model.compile(
        optimizer=SGD(learning_rate=learning_rate, momentum=momentum),
        loss="categorical_crossentropy")

    print("Start training - phase 2...")
    checkpoint = ModelCheckpoint(
        "logs/weights.h5",
        monitor=checkpoint_monitor,
        save_best_only=True,
        save_freq=checkpoint_period_after_unfreeze)
    train(checkpoint, epochs_after_unfreeze)

print("Saving model...")
write_labels(labels_path)
tf.saved_model.save(model, model_path)

if tflite_model_path:
    print("Converting model...")
    converter = tf.lite.TFLiteConverter.from_saved_model(model_path)
    tflite_model = converter.convert()
    tf.lite.experimental.Analyzer.analyze(
        model_content=tflite_model, gpu_compatibility=True)
    print("Saving TFLite model...")
    with open(tflite_model_path + "/saved_model.tflite", "wb") as f:
        f.write(tflite_model)

# end time
end = time.time()
print("[STATUS] end time - {}".format(datetime.datetime.now().strftime("%Y-%m-%d %H:%M")))
print("[STATUS] total duration: {}".format(end - start))

print("Testing...")
test()
