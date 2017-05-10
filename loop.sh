#!/bin/bash


for i in {1..20}
do
echo "job$1"
echo $RANDOM >> random_test
git add .
git commit -m "job $i"
git push origin loop-flaking
sleep 2
done 
