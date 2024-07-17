#!/bin/bash

version=$(grep Version ../main.go  | head -1 | sed 's/^\(.*\)"\(.*\)".*/\2/')
release_date=$(grep Release_date ../main.go  | head -1 | sed 's/^\(.*\)"\(.*\)".*/\2/')

echo "Scanned ../main.go to gather Version and Release date"
echo ""
echo "Version      : $version"
echo "Release date : $release_date"
echo ""

header="---
title: \"pgSimload ${version} Documentation\"
author: [Jean-Paul Argudo]
date: \"${release_date}\"
subject: \"PostgreSQL\"
keywords: [pgSimload, test, HA, PostgreSQL, SQL]
lang: \"en\"
colorlinks: true
toc: true
toc-own-page: true
toc-float: true
toc-depth: 4
numbersections: true
titlepage: true,
titlepage-rule-color: \"FFFFFF\"
titlepage-rule-height: 1
titlepage-text-color: \"FFFFFF\"
titlepage-background: \"cdbackground6.pdf\"
listings-disable-line-numbers: true
header-includes: |
    \usepackage{sectsty}
    \sectionfont{\clearpage}
...
"

readme="# pgSimload $version documentation

## [Overview](01_overview.md)

## [Examples](02_examples.md)

## [Flags and Parameters](03_overview_of_flags_and_parameters.md)

## [Release notes](04_release_notes.md)

## [Roadmap](05_roadmap.md)
"

echo "****************************************************************"
echo "HEADER to build documentation used will be this one"
echo "****************************************************************"
echo "Carefully *VERIFY* VERSION and DATE before accepting"
echo "****************************************************************"
echo "$header"
echo "****************************************************************"
echo ""
read -p "Should we proceed updating the documentation [y/N]: " PROCEED
PROCEED=`echo ${PROCEED:-N} | tr 'a-z' 'A-Z'`

if [[ "${PROCEED}" == "Y" ]]
then
   
  echo "$header" > doc.md
  cat 0*.md >> doc.md

  echo "Compiling PDF..."
  pandoc doc.md -o "pgSimload.doc.pdf" --from markdown --template=eisvogel.tex --listings
  rm doc.md
  echo "pgSimload documentation PDF updated!"
 
  echo ""
  echo "$readme" > README.md
  echo "README.md updated"
 
else
  echo "No worries, getting outta of here!"
fi


