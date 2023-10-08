from sointu import (
    play_song,
    playback_position,
    playback_finished,
    sample_rate,
    track_length,
)
from sys import exit

if __name__ == '__main__':
    play_song()

    while not playback_finished():
        print("Playback time:", playback_position() / sample_rate())

    exit(0)
