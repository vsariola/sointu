import typing
from construct import Struct, Array, Bytes, Int32ul, Enum, Int8ul, Const, Switch, this, Error, Padded, FlagsEnum, PaddedString, Container, ListContainer, EnumIntegerString, EnumInteger
from enum import IntEnum, IntFlag

MAX_POLYPHONY = 2
MAX_INSTRUMENTS = 16
MAX_UNITS = 64
MAX_UNIT_SLOTS = 16
MAX_INSTRUMENT_NAME_LENGTH = 64

DEFAULT_GLOBAL_NAME = b'GlobalUnitsStoredAs.4ki                                         '

class VersionTag(IntEnum or IntFlag):
    VERSION_TAG_10 = 0x30316b34 # 4k10
    VERSION_TAG_11 = 0x31316b34 # 4k11
    VERSION_TAG_12 = 0x32316b34 # 4k12
    VERSION_TAG_13 = 0x33316b34 # 4k13
    VERSION_TAG_CURRENT = 0x34316b34 # 4k14

class UnitId(IntEnum or IntFlag):
    M_NONE = 0x0
    M_ENV = 0x1
    M_VCO = 0x2
    M_VCF = 0x3
    M_DST = 0x4
    M_DLL = 0x5
    M_FOP = 0x6
    M_FST = 0x7
    M_PAN = 0x8
    M_OUT = 0x9
    M_ACC = 0xA
    M_FLD = 0xB
    M_GLITCH = 0xC
    NUM_MODULES = 0xD

class VCOFlags(IntEnum or IntFlag):
    VCO_SINE = 0x01
    VCO_TRISAW = 0x02
    VCO_PULSE = 0x04
    VCO_NOISE = 0x08
    VCO_LFO	= 0x10
    VCO_GATE = 0x20
    VCO_STEREO = 0x40

class VCFType(IntEnum or IntFlag):
    VCF_LOWPASS = 0x1
    VCF_HIGHPASS = 0x2
    VCF_BANDPASS = 0x4
    VCF_BANDSTOP = 0x3
    VCF_ALLPASS = 0x7
    VCF_PEAK = 0x8
    VCF_STEREO = 0x10

class FOPFlags(IntEnum or IntFlag):
    FOP_POP	= 0x1
    FOP_ADDP = 0x2
    FOP_MULP = 0x3
    FOP_PUSH = 0x4
    FOP_XCH	= 0x5
    FOP_ADD	= 0x6
    FOP_MUL	= 0x7
    FOP_ADDP2 = 0x8
    FOP_LOADNOTE = 0x9
    FOP_MULP2 = 0xA

class FSTType(IntEnum or IntFlag):
    FST_SET = 0x00
    FST_ADD = 0x10
    FST_MUL = 0x20
    FST_POP = 0x40

class ACCFlags(IntEnum or IntFlag):
    ACC_OUT = 0x0
    ACC_AUX = 0x8

formatnone = Bytes(MAX_UNIT_SLOTS - 1)

formatenv = Struct(
    "attack" / Int8ul,
    "decay" / Int8ul,
    "sustain" / Int8ul,
    "release" / Int8ul,
    "gain" / Int8ul,
)

formatvco = Struct(
    "transpose" / Int8ul,
    "detune" / Int8ul,
    "phhaseofs" / Int8ul,
    "gate" / Int8ul,
    "color" / Int8ul,
    "shape" / Int8ul,
    "gain" / Int8ul,
    "flags" / FlagsEnum(Int8ul, VCOFlags),
)

# TODO: how do we add this conveniently?
formatvco11 = Struct(
    "transpose" / Int8ul,
    "detune" / Int8ul,
    "phhaseofs" / Int8ul,
    "color" / Int8ul,
    "shape" / Int8ul,
    "gain" / Int8ul,
    "flags" / FlagsEnum(Int8ul, VCOFlags),
)

formatvcf = Struct(
    "freq" / Int8ul,
    "res" / Int8ul,
    "type" / Enum(Int8ul, VCFType),
)

formatdst = Struct(
    "drive" / Int8ul,
    "snhfreq" / Int8ul,
    "stereo" / Int8ul,
)

formatdll = Struct(
    "pregain" / Int8ul,
    "dry" / Int8ul,
    "feedback" / Int8ul,
    "damp" / Int8ul,
    "freq" / Int8ul,
    "depth" / Int8ul,
    "delay" / Int8ul,
    "count" / Int8ul,
    "guidelay" / Int8ul,
    "synctype" / Int8ul, # TODO: Enum? Where?
    "leftreverb" / Int8ul,
    "reverb" / Int8ul,
)

# TODO: Implement the migrations
formatdll10 = Struct(
    "pregain" / Int8ul,
    "dry" / Int8ul,
    "feedback" / Int8ul,
    "damp" / Int8ul,
    "delay" / Int8ul,
    "count" / Int8ul,
    "guidelay" / Int8ul,
    "synctype" / Int8ul, # TODO: Enum? Where?
    "leftreverb" / Int8ul,
    "reverb" / Int8ul,
)

formatfop = Struct(
    "flags"/ FlagsEnum(Int8ul, FOPFlags),
)

formatfst = Struct(
    "amount" / Int8ul,
    "type" / Enum(Int8ul, FSTType),
    "dest_stack" / Int8ul,
    "dest_unit" / Int8ul,
    "dest_slot" / Int8ul,
    "dest_id" / Int8ul,
)

formatpan = Struct(
    "panning" / Int8ul,
)

formatout = Struct(
    "gain" / Int8ul,
    "auxsend" / Int8ul,
)

formatacc = Struct(
    "flags" / FlagsEnum(Int8ul, ACCFlags),
)

formatfld = Struct(
    "value" / Int8ul,
)

formatglitch = Struct(
    "active" / Int8ul,
    "dry" / Int8ul,
    "dsize" / Int8ul,
    "dpitch" / Int8ul,
    "delay" / Int8ul,
    "guidelay" / Int8ul,
)

# TODO: unclear what this does, find out.
formatnummodules = Struct(
    "placeholder" / Int8ul,
)

format4ku = Struct(
    "id" / Enum(Int8ul, UnitId),
    "slots" / Padded(
        MAX_UNIT_SLOTS - 1,
        Switch(
            keyfunc = this.id,
            cases = {
                UnitId.M_ENV.name: formatenv,
                UnitId.M_VCO.name: formatvco,
                UnitId.M_VCF.name: formatvcf,
                UnitId.M_DST.name: formatdst,
                UnitId.M_DLL.name: formatdll,
                UnitId.M_FOP.name: formatfop,
                UnitId.M_FST.name: formatfst,
                UnitId.M_PAN.name: formatpan,
                UnitId.M_OUT.name: formatout,
                UnitId.M_ACC.name: formatacc,
                UnitId.M_FLD.name: formatfld,
                UnitId.M_GLITCH.name: formatglitch,
                UnitId.NUM_MODULES.name: formatnummodules,
                UnitId.M_NONE.name: formatnone,
            },
            default = formatnone,
        ),
    ),
)

format4ki = Struct(
    "versionTag" / Enum(Int32ul, VersionTag),
    "instrumentName" / PaddedString(MAX_INSTRUMENT_NAME_LENGTH, 'utf-8'),
    "units" / Array(MAX_UNITS, format4ku),
)

format4kp = Struct(
    "versionTag" / Enum(Int32ul, VersionTag),
    "polyphony" / Int32ul,
    "instrumentNames" / Array(MAX_INSTRUMENTS, PaddedString(MAX_INSTRUMENT_NAME_LENGTH, 'utf-8')),
    "instrumentValues" / Array(MAX_INSTRUMENTS * MAX_UNITS, format4ku),
    "globalValues" / Array(MAX_UNITS, format4ku),
)

class FormatConverter:
    builtins = ['copy', 'search', 'search_all', 'update']

    @staticmethod
    def ConstructToDictTrivial(construct: typing.Any) -> dict:
        '''
        Convert a construct container to a dictionary for stupid-simple
        json and yaml dumps. Those do not work with sointu, as its json
        and yaml formats are simpler and more structured than 4klang's
        binary format.

            Parameters:
                construct (typing.Any): Construct to convert to dictionary.

            Returns:
                result (dict): json-serializable dictionary.

        '''
        if type(construct) is Container:
            result = {}
            children = list(filter(lambda id: not id.startswith('_') and not id in FormatConverter.builtins, dir(construct)))
            for child in children:
                result[child] = FormatConverter.ConstructToDict(getattr(construct, child))
            return result
        elif type(construct) is ListContainer:
            result = []
            for child in construct:
                result.append(FormatConverter.ConstructToDict(child))
            return result
        elif type(construct) is EnumIntegerString:
            return str(construct)
        elif type(construct) is EnumInteger:
            return int(construct)
        elif type(construct) is int:
            return int(construct)
        elif type(construct) is bytes:
            return construct.decode('utf-8')
        elif type(construct) is bool:
            return construct
        elif type(construct) is str:
            return str(construct).split('\x00')[0]
        
        raise Exception("Unrecognized construct type: {}".format(type(construct)))

    # TODO: It needs to be verified whether those identifiers
    # are the correct ones for sointu.
    @staticmethod
    def UnitTypeName(id: UnitId) -> str:
        if id == UnitId.M_ACC.name:
            return "accumulate"
        elif id == UnitId.M_DLL.name:
            return "delay"
        elif id == UnitId.M_DST.name:
            return "distort"
        elif id == UnitId.M_ENV.name:
            return "envelope"
        elif id == UnitId.M_FLD.name:
            return "load"
        elif id == UnitId.M_FOP.name:
            # TODO: depending on the arithmetic handling in sointu,
            # this might not be the correct approach
            return "arithmetic"
        elif id == UnitId.M_FST.name:
            return "store"
        elif id == UnitId.M_GLITCH.name:
            return "glitch"
        elif id == UnitId.M_OUT.name:
            return "out"
        elif id == UnitId.M_PAN.name:
            return "pan"
        elif id == UnitId.M_VCF.name:
            return "filter"
        elif id == UnitId.M_VCO.name:
            return "oscillator"
        elif id == UnitId.M_NONE.name:
            return "none"
        else:
            return "unsupported"

    @staticmethod
    def UnitContainerToDict(construct: typing.Any) -> dict:
        unitType = FormatConverter.UnitTypeName(construct.id)
        
        slots = {}
        for (name, value) in construct.slots.items():
            if name == '_io':
                continue

            slots[name] = value
                
        return {
            "type": unitType,
            "parameters": slots,
        }

    @staticmethod
    def PatchContainerToDict(construct: typing.Any) -> dict:
        instruments = []
        for instrumentIndex in range(MAX_INSTRUMENTS):
            instrument = []
            for unitIndex in range(MAX_UNIT_SLOTS):
                valueIndex = instrumentIndex*MAX_UNIT_SLOTS + unitIndex

                # Skip empty units.
                if construct.instrumentValues[valueIndex].id == UnitId.M_NONE.name:
                    continue

                unitDict = FormatConverter.UnitContainerToDict(construct.instrumentValues[valueIndex])
                instrument.append(unitDict)

            if instrument != []:
                instruments.append({
                    "numvoices": 1,
                    "units": instrument,
                })
        
        return {
            "patch": instruments
        }

    @staticmethod
    def migrate(fromVersion: VersionTag, toVersion: VersionTag) -> typing.Any:
        return None