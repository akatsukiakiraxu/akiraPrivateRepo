#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <stdarg.h>
#include <stdbool.h>

short getMinMax(double* array, int count, double* min ,double* max)
{
    if(array == NULL || min == NULL || max == NULL){
        return -1;
    }
    for(int i=0;i<count;i++){
        if(*min>array[i]){
            *min = array[i];
        }
        if(*max<array[i]){
            *max = array[i];
        }
    }
    return 0;
}

void ADCDB_DEBUG(bool flag, char *tsv, char *format, ...) {
    if (flag) {
    va_list arg;

    va_start(arg, format);
    vsprintf(tsv, format, arg);
    va_end(arg);
        //fprintf(stderr, param, __FILE__, __FUNCTION__,  __LINE__, __VA_ARGS__ );
    }
    else {
    }
}

int main(){
#if 0
    double userDataBuffer[40];
    int chanCount = 4;
    int chanStart = 0;
    int channelCountMax = 15;
    int dataCount = 40;
    for(int i =0; i < 40; i=i+4){
        userDataBuffer[i] = 0;
        userDataBuffer[i+1] = 1;
        userDataBuffer[i+2] = 2;
        userDataBuffer[i+3] = 3;
    }
    for(int i =0; i < 40; ++i){
        printf("%2.2f, ", userDataBuffer[i]);
    }
    printf("\n");

    double outbuf[16];
    int countPerChannel=dataCount/chanCount;
    double scaleData[chanCount][countPerChannel];
    double expMax = 0xFFFF;
    double expMin = 0;
/*
    for(int j = 0; j < countPerChannel; ++j){
        scaleData[0][j] = userDataBuffer[2+(j*chanCount)];
    }
    for(int j = 0; j < countPerChannel; ++j){
        scaleData[1][j] = userDataBuffer[3+(j*chanCount)];
    }
    for(int j = 0; j < countPerChannel; ++j){
        scaleData[2][j] = userDataBuffer[0+(j*chanCount)];
    }
    for(int j = 0; j < countPerChannel; ++j){
        scaleData[3][j] = userDataBuffer[1+(j*chanCount)];
    }
    */
    
    for(int k = 0; k < chanCount; ++k){
        int startPos =(k>=chanStart)?(k - chanStart):(k+(chanCount-chanStart));
        for(int j = 0; j < countPerChannel; ++j){
            scaleData[k][j] = userDataBuffer[startPos+(j*chanCount)];
        }
    } 
       

    for(int k=0; k < chanCount; ++k){
         printf("ch[%d]:",k);
         for(int m=0; m < countPerChannel; ++m){
             printf("%2.2f,",scaleData[k][m]);
        }
         printf("\n");
    }

    memset(outbuf, 0x00, sizeof(outbuf));
    for(int k=0; k < chanCount; ++k){
        double max=scaleData[k][0],min=scaleData[k][0];
        getMinMax(&scaleData[k][0], countPerChannel, &min, &max);
        outbuf[0+(k*4)] = max;
        outbuf[1+(k*4)] = min;
        outbuf[2+(k*4)] = expMax;
        outbuf[3+(k*4)] = expMin;
    }

    for(int k=0; k < 16; ++k){
         printf("%2.2f, ", outbuf[k]);
    }    
         printf("\n");
#endif
#if 0
    char *filename = "/home/share/adcdb_dumpData";
    FILE *fp, *fpw;
    fp=fopen(filename,"rb");
    fpw=fopen("/home/share/adcdb_dd","w");
    if(fp==NULL){
    return -1;}
    struct stat stat_buf;
    if(stat(filename,&stat_buf)==0){
        printf("%ld\n", stat_buf.st_size);
    }

    int size = 0;
    while (size < stat_buf.st_size) {
        short type = 0;
        fread((void*)&type,1,sizeof(short),fp);
        size += sizeof(short);
        short triggerFlag = 0;
        fread((void*)&triggerFlag,1,sizeof(short),fp);
        size += sizeof(short);
        int len = 0;
        fread((void*)&len,1,sizeof(int),fp);
        size += sizeof(int);
        if (len <= 0) {
            break;
        }
        if (type == 0 && len == 128) {
            double value[4] = {0.0};
            for (int i = 0; i < len; i+=32) {
                fread((void*)&value[0],1,sizeof(double),fp);
                fread((void*)&value[1],1,sizeof(double),fp);
                fread((void*)&value[2],1,sizeof(double),fp);
                fread((void*)&value[3],1,sizeof(double),fp);
                fprintf(fpw,"%2.4f,%2.4f,%2.4f,%2.4f\n",value[0],value[1],value[2],value[3]);
                size += 32;
            }
         }
         else {
            fseek(fp, len, SEEK_CUR);         
         }
    }
    fclose(fpw);
    fclose(fp);
#endif

#if 0
    char *filename = "debug.raw.bin";
    FILE *fp, *fpw;
    fp=fopen(filename,"rb");
    fpw=fopen("adcdb_rawdd","w");
    if(fp==NULL){
    return -1;}
    struct stat stat_buf;
    if(stat(filename,&stat_buf)==0){
        printf("%ld\n", stat_buf.st_size);
    }

    int size = 0;
    while (size < stat_buf.st_size) {
        int len = 0;
        fread((void*)&len,1,sizeof(int),fp);
        size += sizeof(int);
        //    printf("len %d\n", len);
        double channelflag = 0;
        fread((void*)&channelflag,1,sizeof(double),fp);
        size += sizeof(double);
        if (len <= 0 || size > 3*1024*1024) {
            printf("break\n");
            break;
        }
        double value = 0;
        for (int i = 0; i < (len-8); i+=8) {
            fread((void*)&value,1,sizeof(double),fp);
            fprintf(fpw,"%2.4f\n",value);
            size += 8;
        }
    }
    fclose(fpw);
    fclose(fp);
#endif
    char tsv[8] = {'\0'};
    ADCDB_DEBUG(true, tsv, "aaa %d", 1);
    printf("%s\n", tsv);
    return 0;
}
