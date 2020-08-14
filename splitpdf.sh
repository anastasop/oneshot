#!/bin/bash

npages=$(pdfinfo "$1" | grep Pages | awk '{print $2}')
pref=${2:-page}

for ((i=1; i <= ${npages}; i++)) do
  gs -sDEVICE=pdfwrite -q -dNOPAUSE -dBATCH -sOutputFile=${pref}-$i.pdf -dFirstPage=$i -dLastPage=$i "$1" 
done
