# origin: https://github.com/amineHorseman/mobilenet-v2-custom-dataset

# conda create --name tensorflow241
# conda activate tensorflow241
# conda install python=3.6
# conda install opencv
# pip install tensorflow==2.4.1
# pip install keras==2.4
# pip install pillow
# pip install imageio
# pip install scikit-image

# img_size: 224, 192, 160, 128, 96 (see: https://huggingface.co/models?search=mobilenet_v2)
# the number of images of each class must match the batch_size
# class folders must be named as 0, 1, 2, 3, ...
# train_path ->
#               0 ->
#                    *.jpg
#                    *.jpg
#                    *.jpg
#               1 ->
#                    *.jpg
#                    *.jpg
#                    *.jpg
#               2 ->
#                    *.jpg
#                    *.jpg
#                    *.jpg
#                    ...

from imageio import imread
from skimage.transform import resize
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
from keras.utils import to_categorical
from keras.optimizers import SGD
from keras.callbacks import ModelCheckpoint
from keras.preprocessing import image
from keras.preprocessing.image import ImageDataGenerator
from keras.applications.mobilenet_v2 import preprocess_input

warnings.simplefilter(action="ignore", category=FutureWarning)
os.environ["TF_CPP_MIN_LOG_LEVEL"] = "3"

# load the user configs
with open("conf.json") as f:
    config = json.load(f)

# config variables
weights = config["weights"]
img_size = config["img_size"]
learning_rate = config["learning_rate"]
momentum = config["momentum"]
train_path = config["train_path"]
test_path = config["test_path"]
model_path = config["model_path"]
batch_size = config["batch_size"]
epochs = config["epochs"]
classes = config["classes"]
augmented_data = config["augmented_data"]
validation_split = config["validation_split"]
data_augmentation = config["data_augmentation"]
epochs_after_unfreeze = config["epochs_after_unfreeze"]
checkpoint_period = config["checkpoint_period"]
checkpoint_period_after_unfreeze = config["checkpoint_period_after_unfreeze"]


def generate_batches(path, batchSize, classes, start, end):
    x = np.empty((classes*(end-start), img_size, img_size, 3))
    y = np.empty(classes*(end-start), dtype=int)
    n = 0
    files = glob.glob(path + "/*/*jpg")
    for f in range(0, len(files), batchSize):
        for i in range(f+start, f+end):
            if i < len(files):
                img = imread(files[i])
                img = preprocess_input(img)
                x[n] = resize(img, (img_size, img_size))
                y[n] = int(files[i].replace("\\", "/").split("/")[1])
                n += 1
    return (x, to_categorical(y, num_classes=classes))


def generate_batches_with_augmentation(train_path, batch_size, validation_split, augmented_data):
    train_datagen = ImageDataGenerator(
        shear_range=0.2,
        rotation_range=0.3,
        zoom_range=0.1,
        validation_split=validation_split)
    train_generator = train_datagen.flow_from_directory(
        train_path,
        target_size=(img_size, img_size),
        batch_size=batch_size,
        save_to_dir=augmented_data)
    return train_generator


def create_folders(model_path, augmented_data):
    if not os.path.exists(model_path):
        os.mkdir(model_path)
    if not os.path.exists(augmented_data):
        os.mkdir(augmented_data)
    if not os.path.exists("logs"):
        os.mkdir("logs")


print("Tensorflow version: "+tf.__version__)

create_folders(model_path, augmented_data)

# create model
base_model = MobileNetV2(include_top=True, weights=weights,
                         input_tensor=Input(shape=(img_size, img_size, 3)), input_shape=(img_size, img_size, 3))
predictions = Dense(classes, activation="softmax")(
    base_model.layers[-2].output)
model = Model(inputs=base_model.input, outputs=predictions)

print("[INFO] successfully loaded base model and model...")
model.summary()

# create callbacks
checkpoint = ModelCheckpoint(
    "logs/weights.h5", monitor="loss", save_best_only=True, period=checkpoint_period)

# start time
start = time.time()

print("Freezing the base layers. Unfreeze the last layer...")
for layer in model.layers[:-1]:
    layer.trainable = False
model.compile(optimizer="rmsprop", loss="categorical_crossentropy")

print("Start training...")
files = glob.glob(train_path + "/*/*jpg")
samples = len(files)

if data_augmentation:
    model.fit(
        generate_batches_with_augmentation(
            train_path, batch_size, validation_split, augmented_data),
        verbose=1,
        epochs=epochs,
        callbacks=[checkpoint])
else:
    validation_bound = int(batch_size*(1-validation_split))
    train_data = generate_batches(
        train_path, batch_size, classes, 0, validation_bound)
    validation_data = generate_batches(
        train_path, batch_size, classes, validation_bound, batch_size)
    model.fit(
        x=train_data[0],
        y=train_data[1],
        epochs=epochs,
        steps_per_epoch=samples//batch_size,
        verbose=1,
        validation_data=validation_data,
        callbacks=[checkpoint])

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
        monitor="loss",
        save_best_only=True,
        period=checkpoint_period_after_unfreeze)

    if data_augmentation:
        model.fit_generator(
            generate_batches_with_augmentation(
                train_path, batch_size, validation_split, augmented_data),
            verbose=1,
            epochs=epochs,
            callbacks=[checkpoint])
    else:
        validation_bound = int(batch_size*(1-validation_split))
        train_data = generate_batches(
            train_path, batch_size, classes, 0, validation_bound)
        validation_data = generate_batches(
            train_path, batch_size, classes, validation_bound, batch_size)
        model.fit_generator(
            x=train_data[0],
            y=train_data[1],
            epochs=epochs_after_unfreeze,
            steps_per_epoch=samples//batch_size,
            verbose=1,
            validation_data=validation_data,
            callbacks=[checkpoint])

print("Saving...")
tf.saved_model.save(model, model_path)

# end time
end = time.time()
print("[STATUS] end time - {}".format(datetime.datetime.now().strftime("%Y-%m-%d %H:%M")))
print("[STATUS] total duration: {}".format(end - start))

print("Testing...")
folders = [name for name in os.listdir(
    test_path) if os.path.isdir(os.path.join(test_path, name))]
for folder in folders:
    success = 0
    average_confidence = 0
    files = glob.glob(test_path + "/" + folder + "/*.jpg")
    for file in files:
        img = imread(file)
        img = preprocess_input(img)
        img = resize(img, (img_size, img_size))
        x = image.img_to_array(img)
        x = np.expand_dims(x, axis=0)
        y_prob = model.predict(x)
        y_class = y_prob.argmax(axis=-1)
        y_class = y_class[0]
        y_confidence = int(y_prob[0][y_class] * 100)
        if y_class == int(folder):
            success += 1
        average_confidence += y_confidence
    success = int(100*success/len(files))
    average_confidence = int(average_confidence / len(files))
    print("class '{}': success rate = {}% with {}% avg confidence".format(
        folder, success, average_confidence))
