#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <stdarg.h>
#include <stdbool.h>
#include <unistd.h>
#include <math.h>

/*
* マジックナンバー 4byte整数 offset=0
* ヘッダ長 4byte整数 offset=4
* 測定データ長 8byte整数 offset=8
* 測定開始時刻 4byte整数 offset=16
* フラグ 4byte整数 offset=20
* チャネル番号 4byte整数 offset=24
* 品番 4byte整数 offset=28
* パレット番号 4byte整数 offset=32
* 位置番号 4byte整数 offset=36
* データエラーカウント 4byte整数 offset=40
* FFTエラーカウント 4byte整数 offset=44
*/

int main(int argc, char* argv[])
{
    FILE* fpw;
    fpw = fopen("rawData.bin", "wb");
    printf("%ld\n", sizeof(long long));
    uint32_t magicNumber = 0x0ADCDB;
    uint32_t headerSize = 40;
    uint32_t startTime = 0x5AC493F9;
    uint32_t flag = 0;
    uint32_t channel = 0;
    uint32_t code = 0;
    uint32_t palette = 0;
    uint32_t pos = 0;
    uint32_t dataErrCnt = 0;
    uint32_t fftErrCnt = 0;



    fwrite(&magicNumber, sizeof(uint32_t), 1, fpw);
    fwrite(&headerSize, sizeof(uint32_t), 1, fpw);


    int samplingfreq = 22050;
    uint16_t *int16Buf = (uint16_t*)malloc(sizeof(uint16_t)*samplingfreq);
    uint64_t dataSize = sizeof(int16Buf);
    fwrite(&dataSize, sizeof(uint64_t), 1, fpw);
    fwrite(&startTime, sizeof(uint32_t), 1, fpw);
    fwrite(&flag, sizeof(uint32_t), 1, fpw);
    fwrite(&channel, sizeof(uint32_t), 1, fpw);
    fwrite(&code, sizeof(uint32_t), 1, fpw);
    fwrite(&palette, sizeof(uint32_t), 1, fpw);
    fwrite(&pos, sizeof(uint32_t), 1, fpw);
    fwrite(&dataErrCnt, sizeof(uint32_t), 1, fpw);
    fwrite(&fftErrCnt, sizeof(uint32_t), 1, fpw);

    float *pbuff = (float*)malloc(sizeof(float) * samplingfreq);
    float step = 2 * M_PI / samplingfreq;
    for(int i=0; i<samplingfreq; i++){
    	*(pbuff+i) = (0.5 * sin(i * step * 440)) * (1.0 * sin(i * step * 440)) * 100;
    	uint16_t tempvalue = (uint16_t)*(pbuff+i);
    	*(int16Buf+i) = tempvalue;
    }
    fwrite(&int16Buf[0], sizeof(uint16_t), samplingfreq, fpw);
    free(int16Buf);
    free(pbuff);
    fclose(fpw);
    return 0;
}
