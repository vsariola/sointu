import unittest
from os.path import dirname, join
from file_format import *

class TestFileFormat(unittest.TestCase):
    
    # Default4kpFiles = [
        # ,
        # 'baghdad.4kp',
        # 'bf-enlighten.4kp',
        # 'c0c00n_001.4kp',
        # 'dollop.4kp',
        # 'example.4kp',
        # 'example2.4kp',
        # 'kevinspacey.4kp',
        # 'LightRhythm.4kp',
        # 'punqtured-sundowner.4kp',
        # 'untitled2.4kp',
        # 'virgill - 4klang basics.4kp',
    # ],
    @staticmethod
    def load(fileName: str) -> bytes:
        data = b''
        with open(join(dirname(__file__), fileName), 'rb') as f:
            data = f.read()
            f.close()
        return data

    def test_2010_ergon_5(self):
        parsed4kp = format4kp.parse(TestFileFormat.load('2010_ergon_5.4kp'))
        self.assertEqual(parsed4kp.versionTag, VersionTag.VERSION_TAG_12)
        self.assertEqual(parsed4kp.polyphony, 2)
        
        instrumentNames = [
            'Lead',
            'Empty',
            'Strings',
            'Bass',
            'Bassdrum',
            'Snaredrum',
            'Hihat',
            'Control',
            'Instrument 9',
            'Instrument 10',
            'Instrument 11',
            'Instrument 12',
            'Instrument 13',
            'Instrument 14',
            'Instrument 15',
            'Instrument 16',
        ]

        for expectedInstrumentName in instrumentNames:
            self.assertTrue(expectedInstrumentName in parsed4kp.instrumentNames)

        print(parsed4kp)

    # def test_default_4kp_parsing(self):
    #     for default4kpFilename in TestFileFormat.Default4kpFiles:
    #         data = b''
    #         with open(join(dirname(__file__), default4kpFilename), 'rb') as f:
    #             data = f.read()
    #             f.close()
    #         parsed4kp = format4kp.parse(data)
    #         print(parsed4kp)
    #         self.assertEqual(0, 1)

    def test_upper(self):
        self.assertEqual('foo'.upper(), 'FOO')

    def test_isupper(self):
        self.assertTrue('FOO'.isupper())
        self.assertFalse('Foo'.isupper())

    def test_split(self):
        s = 'hello world'
        self.assertEqual(s.split(), ['hello', 'world'])
        # check that s.split fails when the separator is not a string
        with self.assertRaises(TypeError):
            s.split(2)

if __name__ == '__main__':
    unittest.main()
