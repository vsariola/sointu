# Usage: python parse_gmdls.py <path-to-gmdls>
# Parses the GMDLs sample and loop locations and dumps them ready to be included in the BEGIN_SAMPLE_OFFSETS block

import sys

def read_chunk(file,indent=0):
    name = f.read(4)    
    length_bytes = f.read(4)
    length = int.from_bytes(length_bytes,byteorder='little')     
    data = None 
    start = f.tell()
    if name == b"RIFF" or name == b"LIST":        
        name = f.read(4)       
        data = list()            
        while f.tell() < start+length:
            data.append(read_chunk(file,indent + 4))            
        if name == b"wave":
            datablock = next((x[1] for x in data if x[0] == b"data"))                        
            wsmp = next((x[1] for x in data if x[0] == b"wsmp"), None)                        
            if "loopstart" not in wsmp:                                
                loopstart,looplength = datablock["length"]/2-1,1 # For samples without loop, keep on repeating the last sample
            else:
                loopstart,looplength = wsmp["loopstart"],wsmp["looplength"]
            INFO = next((x[1] for x in data if x[0] == b"INFO"), None)            
            INAM = next((x[1] for x in INFO if x[0] == b"INAM"), None)            
            name = ""            
            if INAM is not None:
                name = INAM["name"]
            t = 60 - wsmp["unitynote"] # in MIDI, the middle C = 263 Hz is 60. In Sointu/4klang, it's 72.
            print("SAMPLE_OFFSET START(%d),LOOPSTART(%d),LOOPLENGTH(%d) ; name %s, unitynote %d (transpose to %d), data length %d" % (datablock["start"],loopstart,looplength,name,wsmp["unitynote"],t,datablock["length"]))  
            # Something is oddly off: LOOPSTART + LOOPLENGTH != DATA LENGTH /2, but rather LOOPSTART + LOOPLENGTH != DATA LENGTH /2 - 1
            # Logically, LOOPSTART = 0 would mean the sample loops from the beginning. But then why would they store one extra sample?
            # Or, maybe start+length is the index of the last sample included in the loop. For now, I'm assuming that start+length-1
            # is the last sample and there's one unused sample.
    elif name == b"wsmp":
        f.read(4)
        data = dict()
        data["unitynote"] = int.from_bytes(f.read(2),byteorder='little') 
        f.read(10)
        numloops = int.from_bytes(f.read(4),byteorder='little') 
        if numloops > 0:
            f.read(8)
            data["loopstart"] = int.from_bytes(f.read(4),byteorder='little')
            data["looplength"] = int.from_bytes(f.read(4),byteorder='little')        
    elif name == b"data":
        data = {"start": f.tell()/2, "length": length/2}
    elif name == b"INAM":
        data = {"name":  f.read(length-1).decode("ascii")}        
    f.read(length - f.tell() + start + (length & 1))    
    return (name,data)

if __name__ == "__main__":
    with open(sys.argv[1], "rb") as f:
        read_chunk(f)