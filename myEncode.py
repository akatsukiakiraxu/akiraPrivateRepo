#!/usr/bin/python3.5
# -*- coding: utf-8 -*
import base64
import sys

def print_useage():
    print('         [-----Usage-----]')
    print('#%s [option] [input file] [output file]' % sys.argv[0])
    print('option: -e encode')
    print('        -d decode')
    print('Example: #%s -e foo.txt bar.txt' % sys.argv[0])

def extra_encrypt(orgData):
    #print(orgData)
    encryptData=[]
    for char in orgData:
        encryptChar = ((char&0x0F)<<4) | ((char&0xF0)>>4) 
        #print('%s -> %s' % (hex(char), hex(encryptChar)))
        encryptData.append(encryptChar)
    #print(encryptData)
    return encryptData

if (len(sys.argv) < 4):
    print_useage()
    quit()

option = sys.argv[1]

ld = open(sys.argv[2], 'rb')
contents = ld.read()
ld.close()

fo = open(sys.argv[3], 'wb')

if option == '-e':
    data = base64.b64encode(contents)
    dataList = extra_encrypt(data)
    for c in dataList:
        fo.write(c.to_bytes(1,'big'))
    fo.close()
elif option == '-d':
    tempStr=''
    dataList = extra_encrypt(contents)
    #print(dataList)
    for c in dataList:
        tempStr += c.to_bytes(1,'big').decode('utf-8')
    fo.write(base64.b64decode(tempStr))
    #print(tempStr)
    fo.close()
else:
    print('%s is a bad option!' % option)
quit()
