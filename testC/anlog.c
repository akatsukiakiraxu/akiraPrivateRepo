#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <stdarg.h>
#include <stdbool.h>
#include <unistd.h>
#include <stdint.h>
void help()
{
    printf("./anlog -s/-r/-t -f <infile> -o <outfile>\n");
    printf("-s: summary\n");
    printf("-r: raw\n");
    printf("-t: fft\n");
    printf("-f: infile\n");
    printf("-o: outfile\n");
}

int main(int argc, char* argv[])
{
    int result;
    char infile[128] = {'\0'};
    char outfile[128] = {'\0'};
    enum {
        SUM = 0,
        RAW,
        FFT
    } dataType;
    while ((result = getopt(argc, argv, "srtf:o:")) != -1) {
        switch (result) {
            case 's':
                fprintf(stdout, "dataType = summary\n");
                dataType = SUM;
                break;
            case 'r':
                fprintf(stdout, "dataType = raw\n");
                dataType = RAW;
                break;
            case 't':
                fprintf(stdout, "dataType = fft\n");
                dataType = FFT;
                break;
            case 'f':
                fprintf(stdout, "infile = %s\n", optarg);
                strcpy(infile, optarg);
                break;
            case 'o':
                fprintf(stdout, "outfile = %s\n", optarg);
                strcpy(outfile, optarg);
                break;
            case '?':
                help();
                break;
        }
    }

    FILE* fp, *fpw;
    struct stat stat_buf;
    if (stat(infile, &stat_buf) != 0) {
        printf("bad status of [%s]!\n",infile);
        return -1;
    }
    fp = fopen(infile, "rb");
    fpw = fopen(outfile, "w");
    if (fp == NULL || fpw == NULL) {
        return -1;
    }
    int size = 0;

    if (dataType == SUM) {
        while (size < stat_buf.st_size) {
            short type = 0;
            fread((void*)&type, 1, sizeof(short), fp);
            size += sizeof(short);
            short triggerFlag = 0;
            fread((void*)&triggerFlag, 1, sizeof(short), fp);
            size += sizeof(short);
            int len = 0;
            fread((void*)&len, 1, sizeof(int), fp);
            size += sizeof(int);
            if (len <= 0) {
                break;
            }
            if (type == 0 && len == 128) {
                double value[4] = {0.0};
                for (int i = 0; i < len; i += 32) {
                    fread((void*)&value[0], 1, sizeof(double), fp);
                    fread((void*)&value[1], 1, sizeof(double), fp);
                    fread((void*)&value[2], 1, sizeof(double), fp);
                    fread((void*)&value[3], 1, sizeof(double), fp);
                    fprintf(fpw, "%2.4f,%2.4f,%2.4f,%2.4f\n", value[0], value[1], value[2], value[3]);
                    size += 32;
                }
            } else {
                fseek(fp, len, SEEK_CUR);
            }
        }
    } else if (dataType == RAW) {
        while (size < stat_buf.st_size) {
            /*
            uint32_t len;
            fread((void*)&len, 1, sizeof(len), fp);
    	    printf("len %d\n", len);
            size += sizeof(len);
            double channelflag = 0;
            fread((void*)&channelflag, 1, sizeof(channelflag), fp);
            size += sizeof(channelflag);
            */
            
            uint64_t len;
            char dummy[48];
    	    fread((void*)dummy, 4, sizeof(char), fp);
    	    fread((void*)dummy, 4, sizeof(char), fp);
    	    fread((void*)&len, 1, sizeof(uint64_t), fp);
    	    printf("len %ld\n", len);
    	    fread((void*)dummy, 32, sizeof(char), fp);
    	    size += 48;
    	    if (len <= 0 ) {
                printf("len error\n");
                break;
            }
            //double value = 0;
            uint16_t value = 0;
            //for (int i = 0; i < len-sizeof(channelflag); i += sizeof(value)) {
            for (int i = 0; i < len; i += sizeof(value)) {
                fread((void*)&value, 1, sizeof(value), fp);
                //fprintf(fpw, "%4.6f\n", value);
                fprintf(fpw, "%d\n", value);
                size += sizeof(value);
            }
        }
    }
    fclose(fpw);
    fclose(fp);
    return 0;
}
