#include <stdio.h>
void dump (unsigned char *p, int n);
void main() {
	int i;
	union {
		double d;
		float f;
		unsigned char c[8];
	} uni;

 	printf("number: ");
	scanf("%lf", &uni.d);
	printf("d = %lf\n", uni.d);
	dump(uni.c, 8);
	uni.f = uni.d;
	printf("f = %f\n", uni.f);
	dump(uni.c, 4);
}        
void dump(unsigned char *p, int n) {
	int i;
	
	for (i=n-1;i>=0;i--) printf("%02X ", p[i]);
	printf("\n");
}
