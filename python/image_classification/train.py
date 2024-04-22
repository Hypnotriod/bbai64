# origin: https://github.com/amineHorseman/mobilenet-v2-custom-dataset

# conda create --name tensorflow241
# conda activate tensorflow241
# conda install python=3.6
# pip install tensorflow==2.4.1
# pip install keras==2.4
# pip install scikit-image

# CUDA 11.0: https://developer.nvidia.com/cuda-11.0-download-archive
# cuDNN Library: https://developer.nvidia.com/rdp/cudnn-archive

# img_size: 224, 192, 160, 128, 96 (see: https://huggingface.co/models?search=mobilenet_v2)
# the number of images of each class must match the batch_size
# class folders must be named by class names specified in classes field
# train_data ->
#               class0 ->
#                    *.jpg
#                    *.jpg
#                    *.jpg
#               class1 ->
#                    *.jpg
#                    *.jpg
#                    *.jpg
#               class2 ->
#                    *.jpg
#                    *.jpg
#                    *.jpg
#                    ...

import json
import datetime
import time
import glob
import numpy as np
import os
import warnings
import random
import tensorflow as tf
from skimage import io, transform
from tensorflow import keras
from keras.applications.mobilenet_v2 import MobileNetV2
from keras.models import Model
from keras.layers import Dense, Input
from keras.utils import to_categorical
from keras.optimizers import SGD
from keras.callbacks import ModelCheckpoint
from keras.preprocessing import image
from keras.preprocessing.image import ImageDataGenerator
from keras.applications.mobilenet_v2 import preprocess_input

warnings.simplefilter(action="ignore", category=FutureWarning)
os.environ["TF_CPP_MIN_LOG_LEVEL"] = "3"
os.environ["TF_DISABLE_RZ_CHECK"] = "1"

# load the user configs
with open("conf.json") as f:
    config = json.load(f)

# config variables
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
num_classes = len(classes)
augmented_data = config["augmented_data"]
validation_split = config["validation_split"]
checkpoint_monitor = config["checkpoint_monitor"]
epochs_after_unfreeze = config["epochs_after_unfreeze"]
checkpoint_period = config["checkpoint_period"]
checkpoint_period_after_unfreeze = config["checkpoint_period_after_unfreeze"]

if disable_cuda_devices:
    os.environ["CUDA_VISIBLE_DEVICES"] = "-1"


def generate_batches(path):
    validation_bound = int(batch_size*(1-validation_split))
    x1 = np.empty(
        (num_classes*validation_bound, img_size, img_size, 3))
    y1 = np.empty(num_classes*validation_bound, dtype=int)
    x2 = np.empty(
        (num_classes*(batch_size-validation_bound), img_size, img_size, 3))
    y2 = np.empty(num_classes*(batch_size-validation_bound), dtype=int)
    n1 = 0
    n2 = 0
    files = glob.glob(path + "/*/*")
    if not len(files) == num_classes * batch_size:
        raise Exception(
            "Training data is not equally distributed: check 'classes' or/and 'batch_size' configuration")
    for f in range(0, len(files), batch_size):
        className = files[f].replace("\\", "/").split("/")[1]
        if className not in classes:
            raise Exception(
                "There is no class with: '%s' name found in '%s'" % (className, path))
        classId = classes.index(className)
        xs = []
        for i in range(f, f+batch_size):
            img = io.imread(files[i])
            img = preprocess_input(img)
            xs.append(transform.resize(img, (img_size, img_size)))
        random.shuffle(xs)
        for i in range(0, validation_bound):
            x1[n1] = xs[i]
            y1[n1] = classId
            n1 += 1
        for i in range(validation_bound, batch_size):
            x2[n2] = xs[i]
            y2[n2] = classId
            n2 += 1
    return ((x1, to_categorical(y1, num_classes=num_classes)), (x2, to_categorical(y2, num_classes=num_classes)))


def write_labels(path):
    f = open(path + "/labels.txt", "w")
    f.write("\n".join(classes))
    f.close()


def create_folders():
    if not os.path.exists(model_path):
        os.mkdir(model_path)
    if augmented_data and not os.path.exists(augmented_data):
        os.mkdir(augmented_data)
    if tflite_model_path and not os.path.exists(tflite_model_path):
        os.mkdir(tflite_model_path)
    if not os.path.exists(labels_path):
        os.mkdir(labels_path)
    if not os.path.exists("logs"):
        os.mkdir("logs")


def train(checkpoint, epochs):
    train_data, validation_data = generate_batches(train_path)
    samples = len(train_data[0]) + len(validation_data[0])
    model.fit(
        x=train_data[0],
        y=train_data[1],
        epochs=epochs,
        steps_per_epoch=samples//batch_size,
        verbose=1,
        validation_data=validation_data,
        callbacks=[checkpoint])


def test():
    folders = [name for name in os.listdir(
        test_path) if os.path.isdir(os.path.join(test_path, name))]
    for folder in folders:
        success = 0
        average_confidence = 0
        files = glob.glob(test_path + "/" + folder + "/*")
        for file in files:
            img = io.imread(file)
            img = preprocess_input(img)
            img = transform.resize(img, (img_size, img_size))
            x = image.img_to_array(img)
            x = np.expand_dims(x, axis=0)
            y_prob = model.predict(x)
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


print("Tensorflow version: "+tf.__version__)
create_folders()

# create model
base_model = MobileNetV2(include_top=True, weights=weights,
                         input_tensor=Input(shape=(img_size, img_size, 3)), input_shape=(img_size, img_size, 3))
predictions = Dense(num_classes, activation="softmax")(
    base_model.layers[-2].output)
model = Model(inputs=base_model.input, outputs=predictions)

print("[INFO] Model is successfully loaded and patched...")
model.summary()

# create callbacks
checkpoint = ModelCheckpoint(
    "logs/weights.h5", monitor=checkpoint_monitor, save_best_only=True, period=checkpoint_period)

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
        period=checkpoint_period_after_unfreeze)
    train(checkpoint, epochs_after_unfreeze)

print("Saving model...")
write_labels(labels_path)
tf.saved_model.save(model, model_path)

if tflite_model_path:
    print("Converting model...")
    converter = tf.lite.TFLiteConverter.from_saved_model(model_path)
    tflite_model = converter.convert()
    print("Saving TFLite model...")
    with open(tflite_model_path + "/saved_model.tflite", "wb") as f:
        f.write(tflite_model)

# end time
end = time.time()
print("[STATUS] end time - {}".format(datetime.datetime.now().strftime("%Y-%m-%d %H:%M")))
print("[STATUS] total duration: {}".format(end - start))

print("Testing...")
test()
