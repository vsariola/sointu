import json
import sys
import yaml
from textwrap import indent
from file_format import *

if __name__ == '__main__':
    parsedFileContent = format4kp.parse_file(sys.argv[1])

    sointuFormat = FormatConverter.PatchContainerToDict(parsedFileContent)
    print(yaml.dump(sointuFormat, indent=4))
