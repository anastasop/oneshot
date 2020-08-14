#!/bin/bash

out=$1
shift

gs -sDEVICE=pdfwrite -q -dNOPAUSE -dBATCH -sOutputFile=${out} $*

