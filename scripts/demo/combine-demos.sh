#!/bin/bash
set -e

usage="combine-demos [-s SENDER_DEMO_GIF] [-r RECEIVER_DEMO_GIF] [-o OUTPUT_GIF]
Combine sender and receiver demo gifs into a vertically stacked .gif that preserves quality.
    -s  the sender demo .gif
    -r  the receiver demo .gif
    -o  the combiend output .gif"


options=':hs:r:o:'
while getopts $options option; do
  case "$option" in
    h) echo "$usage"; exit;;
    s) SENDER_DEMO=$OPTARG;;
    r) RECEIVER_DEMO=$OPTARG;;
    o) OUTPUT=$OPTARG;;
    :) printf "missing argument for -%s\n" "$OPTARG" >&2; echo "$usage" >&2; exit 1;;
   \?) printf "illegal option: -%s\n" "$OPTARG" >&2; echo "$usage" >&2; exit 1;;
  esac
done

# enforce arguments
if [ ! "$SENDER_DEMO" ] || [ ! "$RECEIVER_DEMO" ] || [ ! "$OUTPUT" ]; then
  echo "arguments -s, -r and -o must be provided"
  echo "$usage" >&2; exit 1
fi

ffmpeg \
-i $SENDER_DEMO \
-i $RECEIVER_DEMO \
-filter_complex "[0][1]vstack=inputs=2,split[y][z];[y]palettegen[pal];[z][pal]paletteuse" \
$OUTPUT