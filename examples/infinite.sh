#!/bin/bash

seconds=${1:-2}

while :
do
  echo -n "infinite ${seconds}: "
  date
  pwd
  sleep "$seconds"
done
