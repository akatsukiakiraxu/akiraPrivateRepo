#include <stdio.h>
#include <stdlib.h>
#include <math.h>
#include <stdint.h>
#include <time.h>
const int nano2sec = 1000 * 1000 * 1000;

int main(void){
    double pbuff[1024] = {0.0};
    int samplingfreq = 22050;
    int sec = 1;
    int fileSize = samplingfreq * sec;
    double step = 2 * M_PI / samplingfreq;
    int i;

    uint16_t type=3;
    uint16_t trigger = 0;
    uint64_t count = 0;
    uint64_t chmap = 0x0F;
    uint32_t size = 4 * 1024 * sizeof(double) + sizeof(uint64_t);
    /*
    struct timespec tsNow;
    clock_gettime(CLOCK_REALTIME, &tsNow);
    time_t tt = tsNow.tv_sec * nano2sec + tsNow.tv_nsec;
    printf("%10ld\n", tt);
    FILE *fp = fopen("hoge.txt", "w");
    */
    while(1) {
        for(i=0; i<1024; i++){
            *(pbuff+i) = (0.5 * sin(i * step * 440)) * (1.0 * sin(i * step * 440)) * 100;
            //printf("%4.6f\n", *(pbuff+i));
        }
        if (count%10==0) {
            trigger = 1;
        }else {
            trigger = 0;
        }
        fwrite(&type, sizeof(uint16_t), 1, stdout); // write type(2Bytes) to stdout
        fwrite(&trigger, sizeof(uint16_t), 1, stdout); // write tiggerFlag(2Bytes) to stdout
        fwrite(&size, sizeof(uint32_t), 1, stdout);  // write size(4Bytes) to stdout
        fwrite(&chmap, sizeof(uint64_t), 1, stdout); // write channelMap(8Bytes) to stdout
        fwrite(pbuff, sizeof(double), 1024, stdout);
        fwrite(pbuff, sizeof(double), 1024, stdout);
        fwrite(pbuff, sizeof(double), 1024, stdout);
        fwrite(pbuff, sizeof(double), 1024, stdout);
        count++;
    }
    /*
        clock_gettime(CLOCK_REALTIME, &tsNow);
        tt = tsNow.tv_sec * nano2sec + tsNow.tv_nsec;
        printf("%10ld\n", tt);


    fclose(fp);
    */
  return 0;
}
