#!/bin/bash


NLOOP=20

for i in {1..$NLOOP}
do

echo $RANDOM >> random_test
git add .
git commit -m "job $i"
git push origin loop-flaking
sleep 2

done 
