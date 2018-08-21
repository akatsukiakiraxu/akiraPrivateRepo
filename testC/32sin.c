#include <stdio.h>
#include <stdlib.h>
#include <math.h>
//#define USE_STDOUT

int main(void){
  FILE *fpw;
  float *pbuff;
  short *int16Buf;
  int samplingfreq = 22050;
  int sec = 1;
  int fileSize = samplingfreq * sec;
  float step = 2 * M_PI / samplingfreq;
  int i;

  fpw = fopen("32bitF_raw.wav", "wb");
  if (fpw == NULL) exit(EXIT_FAILURE);
  pbuff = (float*)malloc(sizeof(float) * fileSize);
  int16Buf = (short*)malloc(sizeof(short)*fileSize);
  if (pbuff == NULL) exit(EXIT_FAILURE);
  for(i=0; i<fileSize; i++){
	*(pbuff+i) = (0.5 * sin(i * step * 440)) * (1.0 * sin(i * step * 440)) * 100;
	short tempvalue = (short)*(pbuff+i);
	*(int16Buf+i) = tempvalue;
//	printf("%d\t",tempvalue);
}
short type = 0;
//short header[]={0,8,0};
#ifdef USE_STDOUT
//fwrite(header, sizeof(short), sizeof(header)/sizeof(short), stdout);
#endif
//fwrite(header, sizeof(short), sizeof(header)/sizeof(short), fpw);
//char dummy[2]={0};
#ifdef USE_STDOUT
//fwrite(dummy, sizeof(char), sizeof(dummy)/sizeof(char), stdout);
#endif
//fwrite(dummy, sizeof(char), sizeof(dummy)/sizeof(char), fpw);
for (i=0; i < 3; ++i){
 type = type==1? 0 : 1;
#ifdef USE_STDOUT
fwrite(&type, sizeof(short), 1, stdout);
#endif
fwrite(&type, sizeof(short), 1, fpw);
#ifdef USE_STDOUT
fwrite(&fileSize, sizeof(int), 1, stdout);
#endif
fwrite(&fileSize, sizeof(int), 1, fpw);
#ifdef USE_STDOUT
//fwrite(dummy, sizeof(char), sizeof(dummy)/sizeof(char), stdout);
#endif
//fwrite(dummy, sizeof(char), sizeof(dummy)/sizeof(char), fpw);
#ifdef USE_STDOUT
fwrite(int16Buf, sizeof(short), samplingfreq/sizeof(short), stdout);
#endif
fwrite(int16Buf, sizeof(short), samplingfreq/sizeof(short), fpw);
}
  fclose(fpw);
  free(pbuff);
  return 0;
}
