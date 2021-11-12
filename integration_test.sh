#!/bin/bash

GREEN='\033[1;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo
echo -e "${YELLOW}Running go fmt...${NC}"
go fmt ./...

echo
echo -e "${YELLOW}Building application...${NC}"
go build 

echo
echo -e "${YELLOW}Create mock repo with contents:${NC}"
mkdir temp
touch temp/fake.js
touch temp/.git
echo hide >> temp/.gitignore
mkdir temp/hide
mkdir temp/hide/secret
touch temp/hide/shh.txt
echo "###"
# ls -Ra temp
find -L temp
echo "###"
# echo temp/*/*/
# find temp -maxdepth 1 -type df
# ls -alrt temp
# ls -alrt temp/hide

echo
echo -e "${YELLOW}Repoman mock repo with result:${NC}"    
RESULT=$(./repoman temp)
echo ${RESULT}
if echo ${RESULT} | grep -q "[{Extension:js Count:1} {Extension:.gitignore Count:1}]" ; then
    echo -e "${GREEN}PASSED${NC}"
else
    echo -e "${RED}FAILED${NC}"
fi

echo
echo -e "${YELLOW}Delete mock repo${NC}"
rm -rf temp

