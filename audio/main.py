import argparse
from pydub import AudioSegment
import numpy as np
import sys

FN_AVG = 'avg'
FN_MAX = 'max'
FN_AVGMAX = 'avgmax'

ARGS = argparse.ArgumentParser(prog='Analyze mp3')
# ARGS.add_argument('file_in', type=argparse.FileType('r'))
ARGS.add_argument('file_in')
ARGS.add_argument('-b', '--bands', type=int, default=220)
ARGS.add_argument('-f', '--fn', choices=[FN_AVG, FN_MAX, FN_AVGMAX], default=FN_AVG)
# ARGS.add_argument('file_out', type=argparse.FileType('w'))
ARGS.add_argument('-o', '--file_out')
# ARGS.add_argument('-v', '--verbose', type=bool)
ARGS = ARGS.parse_args()

print(ARGS)

print('Loading audio...')
song = AudioSegment.from_file(ARGS.file_in)
# print(len(song.split_to_mono()[0].raw_data))
# d = song.split_to_mono()[0].raw_data
aMono = song.split_to_mono()[0]
aData = aMono.get_array_of_samples()
aDataLen = len(aData)
aDurationSecs = aDataLen / aMono.frame_rate

print("aDurationSecs", aDurationSecs)
print("aDataLen", aDataLen)
print("channels", aMono.channels)
print("sample_width", aMono.sample_width)

class ABandInfo:
  def __init__ (self):
    self.max = 0
    self.avg = 0

aBandList = []
BAND_STEP = round(aDataLen / ARGS.bands)

print("BAND_STEP", BAND_STEP)

print('Analyzing audio..')
for i in range(ARGS.bands):
  aBand = aData[i * BAND_STEP:(i+1) * BAND_STEP]

  aBandInfo = ABandInfo()
  for sample in aBand:
    # print(sample)
    sample = abs(sample)
    aBandInfo.max = max(sample, aBandInfo.max)
    aBandInfo.avg = (sample + aBandInfo.avg) / 2

  writeVal = 0
  if ARGS.fn == FN_AVG:
    writeVal = aBandInfo.avg
  elif ARGS.fn == FN_MAX:
    writeVal = aBandInfo.max
  elif ARGS.fn == FN_AVGMAX:
    writeVal = (aBandInfo.max - aBandInfo.avg) / 2

  print(i, writeVal)
  aBandList.append(writeVal) # !!!!

aBandList.append(aDurationSecs)

FILENAME = f'{ARGS.file_out or ARGS.file_in}.{ARGS.fn}{ARGS.bands}.bin'
print('Writing into: ', FILENAME)

with open(FILENAME, 'wb') as f:
  np.array(aBandList, dtype=np.int16).tofile(f)


print('Done!')