#include <stdio.h>
#include <stdlib.h>
//#include <math.h>
#include <unistd.h>
#include <string.h>
#define GET_ARRAY_COUNT(a) (sizeof(a)/sizeof(a[0]))

const int bufferSize = 38 * 53;
int main(void)
{
    FILE *fpw;
    fpw = fopen("doubleRaw", "wb");
    if (fpw == NULL) exit(EXIT_FAILURE);
    double dd[4] = {1.11111, 5.55555, 3.22222, 4.22222};
    short ss[4] = {5, 6, 7, 8};
    char buff[bufferSize];
    unsigned short type = 0;

    for (int sendCount = 0; sendCount < 99999; ++sendCount) {
        memset(buff, 0x00, bufferSize);
        for (int i = 0; i < 53; ++i) {
            int index = 0;
            *(buff + (38 * i) + (index++)) = 0x00;
            *(buff + (38 * i) + (index++)) = 0x00;

            *(buff + (38 * i) + (index++)) = 0x20;
            *(buff + (38 * i) + (index++)) = 0x00;
            *(buff + (38 * i) + (index++)) = 0x00;
            *(buff + (38 * i) + (index++)) = 0x00;

            /*==============
                CH1
            ===============*/
            //max
            *(buff + (38 * i) + (index++)) = 0x88;
            *(buff + (38 * i) + (index++)) = 0x13;
            //min
            *(buff + (38 * i) + (index++)) = 0x10;
            *(buff + (38 * i) + (index++)) = 0x00;
            //expert max
            *(buff + (38 * i) + (index++)) = 0xFF;
            *(buff + (38 * i) + (index++)) = 0xFF;
            //expert min
            *(buff + (38 * i) + (index++)) = 0x00;
            *(buff + (38 * i) + (index++)) = 0x00;

            /*==============
                CH2
            ===============*/
            //max
            *(buff + (38 * i) + (index++)) = 0x10;
            *(buff + (38 * i) + (index++)) = 0x27;
            //min
            *(buff + (38 * i) + (index++)) = 0x10;
            *(buff + (38 * i) + (index++)) = 0x00;
            //expert max
            *(buff + (38 * i) + (index++)) = 0xFF;
            *(buff + (38 * i) + (index++)) = 0xFF;
            //expert min
            *(buff + (38 * i) + (index++)) = 0x00;
            *(buff + (38 * i) + (index++)) = 0x00;

            /*==============
                CH3
            ===============*/
            //max
            *(buff + (38 * i) + (index++)) = 0x98;
            *(buff + (38 * i) + (index++)) = 0x3A;
            //min
            *(buff + (38 * i) + (index++)) = 0x10;
            *(buff + (38 * i) + (index++)) = 0x00;
            //expert max
            *(buff + (38 * i) + (index++)) = 0xFF;
            *(buff + (38 * i) + (index++)) = 0xFF;
            //expert min
            *(buff + (38 * i) + (index++)) = 0x00;
            *(buff + (38 * i) + (index++)) = 0x00;

            /*==============
                CH4
            ===============*/
            //max
            *(buff + (38 * i) + (index++)) = 0x20;
            *(buff + (38 * i) + (index++)) = 0x4E;
            //min
            *(buff + (38 * i) + (index++)) = 0x10;
            *(buff + (38 * i) + (index++)) = 0x00;
            //expert max
            *(buff + (38 * i) + (index++)) = 0xFF;
            *(buff + (38 * i) + (index++)) = 0xFF;
            //expert min
            *(buff + (38 * i) + (index++)) = 0x00;
            *(buff + (38 * i) + (index++)) = 0x00;
        }
        fwrite(buff, sizeof(char), bufferSize, stdout);
        //fwrite(buff, sizeof(char), bufferSize, fpw);
       //usleep(200000);
    }
   fclose(fpw);
    return 0;
}
