#!/usr/bin/python
#usage: ./caesar.py 'ciphered message'

import sys, array

# needs a dictionary with a word per line
# i used the one from http://swtch.com/usr/local/plan9/lib/words 
words = set()
with open("/usr/local/plan9/lib/words") as f:
    for line in f:
        words.add(line.rstrip().lower())

maxmatches = 0 
deciphered = 'unable to decipher'
for d in xrange(26):
    s = array.array('c', sys.argv[1].lower())
    for i in xrange(len(s)):
        if not s[i].isspace():
            s[i] = chr(ord('a') + (ord(s[i]) - ord('a') + d) % 26)
    candidate = s.tostring()
    nmatches = 0
    for word in candidate.split():
        if word in words:
            nmatches += 1 
    if nmatches > maxmatches:
        deciphered = candidate 
        maxmatches = nmatches

print deciphered