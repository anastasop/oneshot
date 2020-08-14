#!/bin/bash

find ./RawLexar -type f | ruby -n -e 'e=File.extname($_.strip); e = e.empty? ? ".none" : e; puts e.slice(1..-1)' | while read ext ;do mkdir -p LexarDisassembled/${ext} ;done

find ./RawLexar -type f | ruby -n -e 'e=File.extname($_.strip); e = e.empty? ? ".none" : e; puts "mv \"#{$_.strip}\" \"LexarDisassembled/#{e.slice(1..-1)}\""' | /bin/bash
