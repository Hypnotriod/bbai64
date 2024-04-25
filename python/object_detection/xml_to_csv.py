# origin: https://github.com/BenGreenfield825/Tensorflow-Object-Detection-with-Tensorflow-2.0/blob/master/model_training/xml_to_csv.py

# python xml_to_csv.py -f train_data test_data validation_data

import glob
import argparse
import pandas as pd
import xml.etree.ElementTree as ET


def xml_to_csv(path):
    xml_list = []
    for xml_file in glob.glob(path + '/*.xml'):
        tree = ET.parse(xml_file)
        root = tree.getroot()
        for member in root.findall('object'):
            value = (
                root.find('filename').text,
                int(root.find('size').find('width').text),
                int(root.find('size').find('height').text),
                member.find('name').text,
                int(member.find('bndbox').find('xmin').text),
                int(member.find('bndbox').find('ymin').text),
                int(member.find('bndbox').find('xmax').text),
                int(member.find('bndbox').find('ymax').text)
            )
            xml_list.append(value)
    column_name = ['filename', 'width', 'height',
                   'class', 'xmin', 'ymin', 'xmax', 'ymax']
    xml_df = pd.DataFrame(xml_list, columns=column_name)
    return xml_df


parser = argparse.ArgumentParser()
parser.add_argument("-f", "--folders",
                    nargs='+', help="Folders to convert", required=True)
args = parser.parse_args()

for folder in args.folders:
    xml_df = xml_to_csv(folder)
    xml_df.to_csv(folder+'_labels.csv', index=None)
print('Successfully converted xml to csv.')
