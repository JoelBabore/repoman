#!/bin/bash

GREEN='\033[1;32m'
RED='\033[1;31m'
YELLOW='\033[0;33m'
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
mkdir temp/sub
touch temp/sub/fake2.js
echo $'hide\ndummy*.json' >> temp/.gitignore
touch temp/dummy-123.json
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
if echo ${RESULT} | grep -q "[{Extension:js Count:2} {Extension:.gitignore Count:1}]" ; then
    echo -e "${GREEN}PASSED OUTPUT${NC}"
else
    echo -e "${RED}FAILED OUTPUT${NC}"
fi

if echo ${RESULT} | grep -q "skipping temp/hide" \
    && echo ${RESULT} | grep -q "skipping temp/dummy-123.json" \
    && echo ${RESULT} | grep -q "skipping temp/.git" ; then
    echo -e "${GREEN}PASSED SKIP LOGS${NC}"
else
    echo -e "${RED}FAILED SKIP LOGS${NC}"
fi

echo
echo -e "${YELLOW}Delete mock repo${NC}"
rm -rf temp

