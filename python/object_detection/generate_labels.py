
import json

with open("conf.json") as f:
    config = json.load(f)

PB_TEMPLATE = """item {
    id: %s
    name: '%s'
}
"""

labels_path = config["labels_path"]
classes = config["classes"]


def write_labels(path):
    f = open(path + "/labels.txt", "w")
    f.write("\n".join(classes))
    f.close()

    f = open(path + "/labels.pbtxt", "w")
    pbtxt = ""
    for id in range(0, len(classes)):
        pbtxt = pbtxt + PB_TEMPLATE % (id, classes[id])
    f.write(pbtxt)
    f.close()


write_labels(labels_path)
