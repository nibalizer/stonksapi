#!/bin/bash
#
#
#
#
#159 >---// stonksdata.txt from:
#160 >---// ftp://ftp.nasdaqtrader.com/SymbolDirectory/
#161 >---// cat nasdaqlisted.txt otherlisted.txt mfundslist.txt |cut -d "|" -f 1-3 > stonksdata.txt

echo "running"
date
rm -f nasdaqlisted.txt otherlisted.txt mfundslist.txt  stonksdata.txt

echo "ftp commands to run:"
cat ftpcmds.txt

echo ""
echo ""

echo "sending ftp commands"
#ftp -v -n < ftpcmds.txt
lftp -v -f ftpcmds.txt
cat nasdaqlisted.txt otherlisted.txt mfundslist.txt |cut -d "|" -f 1-3 > stonksdata.txt

echo "Created db: stonksdata.txt"
